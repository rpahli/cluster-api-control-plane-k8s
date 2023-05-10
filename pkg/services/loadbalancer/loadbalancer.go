package loadbalancer

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/scope"
)

// Service is a struct with the cluster scope to reconcile load balancers.
type Service struct {
	scope *scope.ClusterScope
}

// NewService creates a new service object.
func NewService(scope *scope.ClusterScope) *Service {
	return &Service{scope: scope}
}

// Reconcile implements the life cycle of HCloud load balancers.
func (s *Service) Reconcile(ctx context.Context) error {
	log := s.scope.Logger.WithValues("reconciler", "load balancer")
	service := &corev1.Service{}
	name := fmt.Sprintf("%s-api-server", s.scope.K8sCluster.Name)
	namespace := "default"
	if s.scope.K8sCluster.Spec.Namespace != "" {
		namespace = s.scope.K8sCluster.Spec.Namespace
	}
	namespaced := types.NamespacedName{Name: name, Namespace: namespace}
	err := s.scope.Client.Get(ctx, namespaced, service)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Create a new loadbalancer")
			service, err = createService(ctx, name, namespace, s.scope.K8sCluster.GetObjectMeta(), log)
			if err := s.scope.Client.Create(ctx, service); err != nil {
				log.Error(err, "Failed to create service")
			}
		}
	}
	s.scope.K8sCluster.Spec.ControlPlaneEndpoint.Host = fmt.Sprintf("%s.default.svc", name)
	s.scope.K8sCluster.Spec.ControlPlaneEndpoint.Port = 6443
	err = s.scope.PatchObject(ctx)
	if err != nil {
		return err
	}
	return nil
}

func createService(ctx context.Context, name, namespace string, object metav1.Object, log logr.Logger) (*corev1.Service, error) {
	or := metav1.NewControllerRef(object,
		infrav1.GroupVersion.WithKind("K8sCluster"))

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port: 6443,
			}},
			Selector: map[string]string{
				"component": "kube-apiserver",
				"tier":      "control-plane",
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
	service.SetOwnerReferences([]metav1.OwnerReference{*or})

	return service, nil
}
