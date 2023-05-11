package controllermanager

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api-provider-nested/controlplane/nested/api/v1beta1"
	kubeadmconstants "sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/remote"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/kubeadm/remote/controlplane"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/scope"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/secret"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Service defines struct with machine scope to reconcile HCloudMachines.
type Service struct {
	scope *scope.ControlPlaneScope
}

// NewControllerManagerService outs a new service with machine scope.
func NewControllerManagerService(scope *scope.ControlPlaneScope) *Service {
	return &Service{
		scope: scope,
	}
}

// Reconcile implements reconcilement of HCloudMachines.
func (s *Service) Reconcile(ctx context.Context) (res reconcile.Result, err error) {

	controlPlaneSecretName := fmt.Sprintf("%s-k8s", s.scope.Cluster.Name)
	s.scope.K8sControlPlane.Spec.ClusterConfiguration.CertificatesDir = "/etc/kubernetes/pki"
	s.scope.K8sControlPlane.Spec.ClusterConfiguration.APIServer.ExtraArgs["etcd-servers"] = "https://pl-etcd:2379"
	podSpecs := controlplane.GetStaticPodSpecs(&s.scope.K8sControlPlane.Spec.ClusterConfiguration, &bootstrapv1.APIEndpoint{
		AdvertiseAddress: "0.0.0.0",
		BindPort:         6443,
	}, controlPlaneSecretName)

	err = s.createOrUpdateApiServerCertificates(ctx)
	if err != nil {
		return res, err
	}
	kubeAPIServerPodSpec := podSpecs[kubeadmconstants.KubeControllerManager]
	kubeAPIServerPodSpec.Name = fmt.Sprintf("%s-%s", s.scope.K8sControlPlane.Name, kubeadmconstants.KubeControllerManager)
	kubeAPIServerPodSpec.Namespace = s.scope.Namespace

	kubeAPIServerPodSpec.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sControlPlane"))})
	res, err = s.createPod(ctx, &kubeAPIServerPodSpec)
	if err == nil {
		return res, nil
	}
	return res, nil
}

func (s *Service) createOrUpdateApiServerCertificates(ctx context.Context) error {
	controlPlaneSecretName := fmt.Sprintf("%s-controller-manager-kubeconfig", s.scope.Cluster.Name)
	namespaced := types.NamespacedName{Name: controlPlaneSecretName, Namespace: s.scope.Namespace}
	sec := &corev1.Secret{}
	err := s.scope.ManagerClient.Get(ctx, namespaced, sec)
	isNew := false
	if err != nil {
		// return error if error not equal not found
		if apierrors.IsNotFound(err) {
			isNew = true
		} else {
			return err
		}
	}
	if !isNew {
		return nil
	}
	configSecret, err := secret.GetFromNamespacedName(ctx, s.scope.ManagerClient, util.ObjectKey(s.scope.Cluster), secret.Kubeconfig)

	controllerManagerSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      controlPlaneSecretName,
			Namespace: s.scope.Namespace,
		},
		Data:       map[string][]byte{},
		StringData: map[string]string{},
		Type:       "",
	}

	controllerManagerSecret.Data[kubeadmconstants.ControllerManagerKubeConfigFileName] = configSecret.Data[secret.KubeconfigDataName]
	controllerManagerSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(s.scope.K8sControlPlane, controlplanev1.GroupVersion.WithKind("K8sControlPlane"))})
	err = s.scope.ManagerClient.Create(ctx, controllerManagerSecret)
	if err != nil {
		return err
	}

	return nil

}

func (s *Service) createPod(ctx context.Context, pod *corev1.Pod) (res reconcile.Result, err error) {
	s.scope.SetReady(true)
	conditions.MarkTrue(s.scope.K8sControlPlane, infrav1.InstanceReadyCondition)
	err = s.scope.ManagerClient.Create(ctx, pod)
	return res, err
}
