package scope

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterScopeParams defines the input parameters used to create a new scope.
type ClusterScopeParams struct {
	Client     client.Client
	APIReader  client.Reader
	Logger     logr.Logger
	K8sSecret  *corev1.Secret
	Cluster    *clusterv1.Cluster
	K8sCluster *infrav1.K8sCluster
}

// NewClusterScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewClusterScope(ctx context.Context, params ClusterScopeParams) (*ClusterScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}
	if params.K8sCluster == nil {
		return nil, errors.New("failed to generate new scope from nil HetznerCluster")
	}
	if params.APIReader == nil {
		return nil, errors.New("failed to generate new scope from nil APIReader")
	}

	emptyLogger := logr.Logger{}
	if params.Logger == emptyLogger {
		return nil, errors.New("failed to generate new scope from nil Logger")
	}

	helper, err := patch.NewHelper(params.K8sCluster, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	return &ClusterScope{
		Logger:        params.Logger,
		Client:        params.Client,
		APIReader:     params.APIReader,
		Cluster:       params.Cluster,
		K8sCluster:    params.K8sCluster,
		patchHelper:   helper,
		hetznerSecret: params.K8sSecret,
	}, nil
}

// ClusterScope defines the basic context for an actuator to operate upon.
type ClusterScope struct {
	logr.Logger
	Client        client.Client
	APIReader     client.Reader
	patchHelper   *patch.Helper
	hetznerSecret *corev1.Secret

	Cluster    *clusterv1.Cluster
	K8sCluster *infrav1.K8sCluster
}

// Name returns the HetznerCluster name.
func (s *ClusterScope) Name() string {
	return s.K8sCluster.Name
}

// Namespace returns the namespace name.
func (s *ClusterScope) Namespace() string {
	return s.K8sCluster.Namespace
}

// HetznerSecret returns the hetzner secret.
func (s *ClusterScope) HetznerSecret() *corev1.Secret {
	return s.hetznerSecret
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ClusterScope) Close(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.K8sCluster)
}

// ControlPlaneAPIEndpointPort returns the Port of the Kube-api server.
func (s *ClusterScope) ControlPlaneAPIEndpointPort() int32 {
	return int32(0) // s.K8sCluster.Spec.ControlPlaneLoadBalancer.Port)
}

// IsControlPlaneReady returns if a machine is a control-plane.
func IsControlPlaneReady(ctx context.Context, c clientcmd.ClientConfig) error {
	restConfig, err := c.ClientConfig()
	if err != nil {
		return err
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	_, err = clientSet.Discovery().RESTClient().Get().AbsPath("/readyz").DoRaw(ctx)
	return err
}
