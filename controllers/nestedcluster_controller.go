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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"
	"time"

	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/v1beta1"
)

// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters,verbs=get;list;watch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8sclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8sclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8sclusters/finalizers,verbs=update

// K8sClusterReconciler reconciles a K8sCluster object.
type K8sClusterReconciler struct {
	client.Client
	Log                            logr.Logger
	WatchFilterValue               string
	targetClusterManagersStopCh    map[types.NamespacedName]chan struct{}
	Scheme                         *runtime.Scheme
	targetClusterManagersLock      sync.Mutex
	TargetClusterManagersWaitGroup *sync.WaitGroup
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := log.FromContext(ctx)

	if r.targetClusterManagersStopCh == nil {
		r.targetClusterManagersStopCh = make(map[types.NamespacedName]chan struct{})
	}

	controller, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.K8sCluster{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		WithEventFilter(predicates.ResourceIsNotExternallyManaged(log)).
		Owns(&corev1.Secret{}).
		Build(r)
	if err != nil {
		return errors.Wrap(err, "error creating controller")
	}

	return controller.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
			c, ok := o.(*clusterv1.Cluster)
			if !ok {
				panic(fmt.Sprintf("Expected a Cluster but got a %T", o))
			}

			log := log.WithValues("objectMapper", "clusterToNestedCluster", "namespace", c.Namespace, "cluster", c.Name)

			// Don't handle deleted clusters
			if !c.ObjectMeta.DeletionTimestamp.IsZero() {
				log.V(1).Info("Cluster has a deletion timestamp, skipping mapping.")
				return nil
			}

			// Make sure the ref is set
			if c.Spec.InfrastructureRef == nil {
				log.V(1).Info("Cluster does not have an InfrastructureRef, skipping mapping.")
				return nil
			}

			if c.Spec.InfrastructureRef.GroupVersionKind().Kind != "K8sCluster" {
				log.V(1).Info("Cluster has an InfrastructureRef for a different type, skipping mapping.")
				return nil
			}

			nc := &infrav1.K8sCluster{}
			key := types.NamespacedName{Namespace: c.Spec.InfrastructureRef.Namespace, Name: c.Spec.InfrastructureRef.Name}

			if err := r.Get(ctx, key, nc); err != nil {
				log.V(1).Error(err, "Failed to get K8sCluster")
				return nil
			}

			if annotations.IsExternallyManaged(c) {
				log.V(4).Info("Nested cluster is externally managed, skipping mapping.")
				return nil
			}

			log.V(1).Info("Adding request.", "nestedCluster", c.Spec.InfrastructureRef.Name)

			return []ctrl.Request{
				{
					NamespacedName: client.ObjectKey{Namespace: c.Namespace, Name: c.Spec.InfrastructureRef.Name},
				},
			}
		}),
	)

}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the K8sCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *K8sClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	// Fetch the HetznerCluster instance
	hetznerCluster := &infrav1.K8sCluster{}
	err := r.Get(ctx, req.NamespacedName, hetznerCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log = log.WithValues("K8sCluster", klog.KObj(hetznerCluster))

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, hetznerCluster.ObjectMeta)
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

	if annotations.IsPaused(cluster, hetznerCluster) {
		log.Info("HetznerCluster or linked Cluster is marked as paused. Won't reconcileNormal")
		return reconcile.Result{}, nil
	}

	if !hetznerCluster.Status.Ready {
		hetznerCluster.Status.Ready = true
		if err := r.Status().Update(ctx, hetznerCluster); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return reconcile.Result{}, nil
}
