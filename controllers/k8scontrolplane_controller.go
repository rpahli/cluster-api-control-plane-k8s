/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package controllers contains all the Infrastructure group controllers for
// running nested clusters.
package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/scope"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/services/apiserver"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/services/controllermanager"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/services/scheduler"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/remote"
	kcpv1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/certs"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/kubeconfig"
	"sigs.k8s.io/cluster-api/util/predicates"
	"sigs.k8s.io/cluster-api/util/secret"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sync"
	"time"

	controlplanev1 "sigs.k8s.io/cluster-api-provider-nested/api/controlplane/v1beta1"
)

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8controlplane,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8controlplane/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8controlplane/finalizers,verbs=update

// K8sControlPlaneReconciler reconciles a K8sControlPlane object.
type K8sControlPlaneReconciler struct {
	client.Client
	Tracker                        *remote.ClusterCacheTracker
	APIReader                      client.Reader
	Log                            logr.Logger
	WatchFilterValue               string
	targetClusterManagersStopCh    map[types.NamespacedName]chan struct{}
	Scheme                         *runtime.Scheme
	targetClusterManagersLock      sync.Mutex
	TargetClusterManagersWaitGroup *sync.WaitGroup
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sControlPlaneReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&controlplanev1.K8sControlPlane{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Complete(r)
}

func (r *K8sControlPlaneReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the K8sCluster instance
	k8sControlPlane := &controlplanev1.K8sControlPlane{}
	err := r.Get(ctx, req.NamespacedName, k8sControlPlane)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log = log.WithValues("K8sControlPlane", klog.KObj(k8sControlPlane))

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, k8sControlPlane.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get owner cluster: %w", err)
	}

	log = log.WithValues("Cluster", klog.KObj(cluster))
	ctx = ctrl.LoggerInto(ctx, log)

	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{
			RequeueAfter: 2 * time.Second,
		}, nil
	}

	if annotations.IsPaused(cluster, k8sControlPlane) {
		log.Info("K8sCluster or linked Cluster is marked as paused. Won't reconcileNormal")
		return reconcile.Result{}, nil
	}

	remoteClient, err := r.Tracker.GetClient(ctx, util.ObjectKey(cluster))
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create remote client: %w", err)
	}

	clusterScope, err := scope.NewControlPlaneScope(ctx, scope.ControlPlaneScopeParams{
		ManagerClient:   r.Client,
		ClusterClient:   remoteClient,
		APIReader:       r.APIReader,
		Logger:          log,
		Cluster:         cluster,
		Namespace:       "default",
		K8sControlPlane: k8sControlPlane,
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create scope: %w", err)
	}

	// Always close the scope when exiting this function so we can persist any K8sCluster changes.
	defer func() {
		if err := clusterScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// check whether rate limit has been reached and if so, then wait.
	if wait := reconcileRateLimit(k8sControlPlane); wait {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Handle deleted clusters
	if !k8sControlPlane.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, clusterScope)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, clusterScope)
}

func (r *K8sControlPlaneReconciler) reconcileNormal(ctx context.Context, controlPlaneScope *scope.ControlPlaneScope) (ctrl.Result, error) {
	k8sControlPlane := controlPlaneScope.K8sControlPlane
	// If the K8sCluster doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(k8sControlPlane, infrav1.ClusterFinalizer)
	if err := controlPlaneScope.PatchObject(ctx); err != nil {
		return ctrl.Result{}, err
	}

	certificates := secret.NewCertificatesForInitialControlPlane(nil)
	controllerRef := metav1.NewControllerRef(controlPlaneScope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("NestedControlPlane"))
	if err := certificates.LookupOrGenerate(ctx, r.Client, util.ObjectKey(controlPlaneScope.Cluster), *controllerRef); err != nil {
		r.Log.Error(err, "unable to lookup or create cluster certificates")
		conditions.MarkFalse(controlPlaneScope.K8sControlPlane, kcpv1.CertificatesAvailableCondition, kcpv1.CertificatesGenerationFailedReason, clusterv1.ConditionSeverityWarning, err.Error())
		return ctrl.Result{}, err
	}
	conditions.MarkTrue(controlPlaneScope.K8sControlPlane, kcpv1.CertificatesAvailableCondition)

	// If ControlPlaneEndpoint is not set, return early
	if !controlPlaneScope.Cluster.Spec.ControlPlaneEndpoint.IsValid() {
		r.Log.Info("Cluster does not yet have a ControlPlaneEndpoint defined")
		return ctrl.Result{}, nil
	}

	if result, err := r.reconcileKubeconfig(ctx, controlPlaneScope); !result.IsZero() || err != nil {
		if err != nil {
			r.Log.Error(err, "failed to reconcile Kubeconfig")
		}
		return result, err
	}

	result, err := apiserver.NewApiServerService(controlPlaneScope).Reconcile(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to reconcile server for K8sControlPlane api server %s/%s: %w",
			controlPlaneScope.K8sControlPlane.Namespace, controlPlaneScope.K8sControlPlane.Name, err)
	}

	result, err = scheduler.NewControllerManagerService(controlPlaneScope).Reconcile(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to reconcile server for K8sControlPlane scheduler %s/%s: %w",
			controlPlaneScope.K8sControlPlane.Namespace, controlPlaneScope.K8sControlPlane.Name, err)
	}

	result, err = controllermanager.NewControllerManagerService(controlPlaneScope).Reconcile(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to reconcile server for K8sControlPlane controller manager %s/%s: %w",
			controlPlaneScope.K8sControlPlane.Namespace, controlPlaneScope.K8sControlPlane.Name, err)
	}
	// create service with load balancer
	return reconcile.Result{}, nil
}

// reconcileKubeconfig will check if the control plane endpoint has been set
// and if so it will generate the KUBECONFIG or regenerate if it's expired.
func (r *K8sControlPlaneReconciler) reconcileKubeconfig(ctx context.Context, controlPlaneScope *scope.ControlPlaneScope) (reconcile.Result, error) {
	endpoint := controlPlaneScope.Cluster.Spec.ControlPlaneEndpoint
	if endpoint.IsZero() {
		return ctrl.Result{}, nil
	}
	controllerOwnerRef := *metav1.NewControllerRef(controlPlaneScope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("NestedControlPlane"))
	clusterName := util.ObjectKey(controlPlaneScope.Cluster)
	configSecret, err := secret.GetFromNamespacedName(ctx, r.Client, clusterName, secret.Kubeconfig)
	switch {
	case apierrors.IsNotFound(err):
		createErr := kubeconfig.CreateSecretWithOwner(
			ctx,
			r.Client,
			clusterName,
			endpoint.String(),
			controllerOwnerRef,
		)
		if errors.Is(createErr, kubeconfig.ErrDependentCertificateNotFound) {
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		// always return if we have just created in order to skip rotation checks
		return ctrl.Result{}, createErr
	case err != nil:
		return ctrl.Result{}, errors.Wrap(err, "failed to retrieve kubeconfig Secret")
	}

	// only do rotation on owned secrets
	if !util.IsControlledBy(configSecret, controlPlaneScope.K8sControlPlane) {
		return ctrl.Result{}, nil
	}

	needsRotation, err := kubeconfig.NeedsClientCertRotation(configSecret, certs.ClientCertificateRenewalDuration)
	if err != nil {
		return ctrl.Result{}, err
	}

	if needsRotation {
		r.Log.Info("rotating kubeconfig secret")
		if err := kubeconfig.RegenerateSecret(ctx, r.Client, configSecret); err != nil {
			return ctrl.Result{}, errors.Wrap(err, "failed to regenerate kubeconfig")
		}
	}

	return ctrl.Result{}, nil
}
func (r *K8sControlPlaneReconciler) reconcileDelete(ctx context.Context, clusterScope *scope.ControlPlaneScope) (reconcile.Result, error) {
	// Cluster is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(clusterScope.K8sControlPlane, infrav1.ClusterFinalizer)

	return reconcile.Result{}, nil
}
