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

package controllers

import (
	"context"
	"fmt"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	"sigs.k8s.io/cluster-api/util/predicates"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/certs"
	"sigs.k8s.io/cluster-api/util/secret"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controlplanev1 "sigs.k8s.io/cluster-api-provider-nested/controlplane/nested/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-nested/controlplane/nested/certificate"
	"sigs.k8s.io/cluster-api-provider-nested/controlplane/nested/kubeadm"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// NestedAPIServerReconciler reconciles a K8sAPIServer object.
type NestedAPIServerReconciler struct {
	client.Client
	Log              logr.Logger
	Scheme           *runtime.Scheme
	WatchFilterValue string
}

// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=k8sapiservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=k8sapiservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=controlplane.cluster.x-k8s.io,resources=k8sapiservers/finalizers,verbs=update

func (r *NestedAPIServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("nestedapiserver", req.NamespacedName)
	log.Info("Reconciling K8sAPIServer...")
	var nkas controlplanev1.K8sAPIServer
	if err := r.Get(ctx, req.NamespacedName, &nkas); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("creating K8sAPIServer",
		"namespace", nkas.GetNamespace(),
		"name", nkas.GetName())

	// 1. check if the ownerreference has been set by the
	// K8sControlPlane controller.
	owner := getOwner(nkas.ObjectMeta)
	if owner == (metav1.OwnerReference{}) {
		// requeue the request if the owner K8sControlPlane has
		// not been set yet.
		log.Info("the owner has not been set yet, will retry later",
			"namespace", nkas.GetNamespace(),
			"name", nkas.GetName())
		return ctrl.Result{Requeue: true}, nil
	}

	var ncp controlplanev1.K8sControlPlane
	if err := r.Get(ctx, types.NamespacedName{Namespace: nkas.GetNamespace(), Name: owner.Name}, &ncp); err != nil {
		log.Info("the owner could not be found, will retry later",
			"namespace", nkas.GetNamespace(),
			"name", owner.Name)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	cluster, err := ncp.GetOwnerCluster(ctx, r.Client)
	if err != nil || cluster == nil {
		log.Error(err, "Failed to retrieve owner Cluster from the control plane")
		return ctrl.Result{}, err
	}

	// 2. create the K8sAPIServer StatefulSet if not found
	nkasName := fmt.Sprintf("%s-apiserver", cluster.GetName())
	var nkasSts appsv1.StatefulSet
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: nkas.GetNamespace(),
		Name:      nkasName,
	}, &nkasSts); err != nil {
		if apierrors.IsNotFound(err) {
			// as the statefulset is not found, mark the K8sAPIServer as unready.
			if IsComponentReady(nkas.Status.CommonStatus) {
				nkas.Status.Phase =
					string(controlplanev1.Unready)
				log.V(5).Info("The corresponding statefulset is not found, " +
					"will mark the K8sAPIServer as unready")
				if err := r.Status().Update(ctx, &nkas); err != nil {
					log.Error(err, "fail to update the status of the K8sAPIServer Object")
					return ctrl.Result{}, err
				}
			}
			if err := r.createAPIServerClientCrts(ctx, cluster, &ncp, &nkas); err != nil {
				log.Error(err, "fail to create K8sAPIServer ManagerClient Certs")
				return ctrl.Result{}, err
			}

			// the statefulset is not found, create one.
			if err := createNestedComponentSts(ctx,
				r.Client, nkas.ObjectMeta, nkas.Spec.NestedComponentSpec,
				controlplanev1.APIServer,
				kubeadm.APIServer, cluster.GetName(), log); err != nil {
				log.Error(err, "fail to create K8sAPIServer StatefulSet")
				return ctrl.Result{}, err
			}
			log.Info("successfully create the K8sAPIServer StatefulSet")
			return ctrl.Result{}, nil
		}
		log.Error(err, "fail to get K8sAPIServer StatefulSet")
		return ctrl.Result{}, err
	}

	// 3. reconcile the K8sAPIServer based on the status of the StatefulSet.
	// Mark the K8sAPIServer as Ready if the StatefulSet is ready.
	if nkasSts.Status.ReadyReplicas == nkasSts.Status.Replicas {
		log.Info("The K8sAPIServer StatefulSet is ready")
		if !IsComponentReady(nkas.Status.CommonStatus) {
			// As the K8sAPIServer StatefulSet is ready, update
			// K8sAPIServer status.
			nkas.Status.Phase = string(controlplanev1.Ready)
			objRef, err := genAPIServerSvcRef(r.Client, nkas, cluster.GetName())
			if err != nil {
				log.Error(err, "fail to generate K8sAPIServer Service Reference")
				return ctrl.Result{}, err
			}
			nkas.Status.APIServerService = &objRef

			log.V(5).Info("The corresponding statefulset is ready, " +
				"will mark the K8sAPIServer as ready")
			if err := r.Status().Update(ctx, &nkas); err != nil {
				log.Error(err, "fail to update K8sAPIServer Object")
				return ctrl.Result{}, err
			}
			log.Info("Successfully set the K8sAPIServer object to ready")
		}
		return ctrl.Result{}, nil
	}

	// mark the K8sAPIServer as unready, if the K8sAPIServer
	// StatefulSet is unready.
	if IsComponentReady(nkas.Status.CommonStatus) {
		nkas.Status.Phase = string(controlplanev1.Unready)
		if err := r.Status().Update(ctx, &nkas); err != nil {
			log.Error(err, "fail to update K8sAPIServer Object")
			return ctrl.Result{}, err
		}
		log.Info("Successfully set the K8sAPIServer object to unready")
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NestedAPIServerReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	_, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&controlplanev1.K8sAPIServer{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("HCloudMachine"))),
		).
		Owns(&appsv1.StatefulSet{}).
		Build(r)
	if err != nil {
		return fmt.Errorf("error creating controller: %w", err)
	}
	return nil
}

// createAPIServerClientCrts will find of create client certs for the etcd cluster.
func (r *NestedAPIServerReconciler) createAPIServerClientCrts(ctx context.Context, cluster *clusterv1.Cluster, ncp *controlplanev1.K8sControlPlane, nkas *controlplanev1.K8sAPIServer) error {
	certificates := secret.NewCertificatesForInitialControlPlane(nil)
	if err := certificates.Lookup(ctx, r.Client, util.ObjectKey(cluster)); err != nil {
		return err
	}
	cacert := certificates.GetByPurpose(secret.ClusterCA)
	if cacert == nil {
		return fmt.Errorf("could not fetch ClusterCA")
	}

	cacrt, err := certs.DecodeCertPEM(cacert.KeyPair.Cert)
	if err != nil {
		return err
	}

	cakey, err := certs.DecodePrivateKeyPEM(cacert.KeyPair.Key)
	if err != nil {
		return err
	}

	// TODO(christopherhein) figure out how to get service clusterIPs.
	apiKeyPair, err := certificate.NewAPIServerCrtAndKey(&certificate.KeyPair{Cert: cacrt, Key: cakey}, nkas.GetName(), "", cluster.Spec.ControlPlaneEndpoint.Host)
	if err != nil {
		return err
	}

	kubeletKeyPair, err := certificate.NewAPIServerKubeletClientCertAndKey(&certificate.KeyPair{Cert: cacrt, Key: cakey}, cluster.Namespace)
	if err != nil {
		return err
	}

	fpcert := certificates.GetByPurpose(secret.FrontProxyCA)
	if fpcert == nil {
		return fmt.Errorf("could not fetch FrontProxyCA")
	}

	fpcrt, err := certs.DecodeCertPEM(fpcert.KeyPair.Cert)
	if err != nil {
		return err
	}

	fpkey, err := certs.DecodePrivateKeyPEM(fpcert.KeyPair.Key)
	if err != nil {
		return err
	}

	frontProxyKeyPair, err := certificate.NewFrontProxyClientCertAndKey(&certificate.KeyPair{Cert: fpcrt, Key: fpkey})
	if err != nil {
		return err
	}

	certs := &certificate.KeyPairs{
		apiKeyPair,
		kubeletKeyPair,
		frontProxyKeyPair,
	}

	controllerRef := metav1.NewControllerRef(ncp, controlplanev1.GroupVersion.WithKind("K8sControlPlane"))
	return certs.LookupOrSave(ctx, r.Client, util.ObjectKey(cluster), *controllerRef)
}
