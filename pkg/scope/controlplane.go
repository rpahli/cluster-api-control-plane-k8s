package scope

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	controlplanev1 "sigs.k8s.io/cluster-api-provider-nested/api/controlplane/v1beta1"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ControlPlaneScopeParams defines the input parameters used to create a new scope.
type ControlPlaneScopeParams struct {
	ManagerClient   client.Client
	ClusterClient   client.Client
	APIReader       client.Reader
	Logger          logr.Logger
	K8sSecret       *corev1.Secret
	Cluster         *clusterv1.Cluster
	K8sCluster      *infrav1.K8sCluster
	Namespace       string
	K8sControlPlane *controlplanev1.K8sControlPlane
}

// NewControlPlaneScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewControlPlaneScope(ctx context.Context, params ControlPlaneScopeParams) (*ControlPlaneScope, error) {
	if params.Cluster == nil {
		return nil, errors.New("failed to generate new scope from nil Cluster")
	}

	if params.APIReader == nil {
		return nil, errors.New("failed to generate new scope from nil APIReader")
	}

	emptyLogger := logr.Logger{}
	if params.Logger == emptyLogger {
		return nil, errors.New("failed to generate new scope from nil Logger")
	}

	helper, err := patch.NewHelper(params.K8sControlPlane, params.ManagerClient)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	return &ControlPlaneScope{
		Logger:          params.Logger,
		ManagerClient:   params.ManagerClient,
		ClusterClient:   params.ClusterClient,
		APIReader:       params.APIReader,
		Cluster:         params.Cluster,
		patchHelper:     helper,
		Namespace:       params.Namespace,
		K8sControlPlane: params.K8sControlPlane,
	}, nil
}

// ControlPlaneScope defines the basic context for an actuator to operate upon.
type ControlPlaneScope struct {
	logr.Logger
	ManagerClient   client.Client
	ClusterClient   client.Client
	APIReader       client.Reader
	patchHelper     *patch.Helper
	Namespace       string
	Cluster         *clusterv1.Cluster
	K8sControlPlane *controlplanev1.K8sControlPlane
}

// SetReady sets the ready field on the machine.
func (m *ControlPlaneScope) SetReady(ready bool) {
	m.K8sControlPlane.Status.Ready = ready
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *ControlPlaneScope) Close(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.K8sControlPlane)
}

// PatchObject persists the machine spec and status.
func (s *ControlPlaneScope) PatchObject(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.K8sControlPlane)
}
