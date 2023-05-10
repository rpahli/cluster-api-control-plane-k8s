package scope

import (
	"context"
	"errors"
	"fmt"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
)

// MachineScopeParams defines the input parameters used to create a new Scope.
type MachineScopeParams struct {
	ClusterScopeParams
	Machine    *clusterv1.Machine
	K8sMachine *infrav1.K8sMachine
	Namespace  string
}

// NewMachineScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewMachineScope(ctx context.Context, params MachineScopeParams) (*MachineScope, error) {
	if params.Machine == nil {
		return nil, errors.New("failed to generate new scope from nil Machine")
	}
	if params.K8sMachine == nil {
		return nil, errors.New("failed to generate new scope from nil HCloudMachine")
	}

	cs, err := NewClusterScope(ctx, params.ClusterScopeParams)
	if err != nil {
		return nil, fmt.Errorf("failed create new cluster scope: %w", err)
	}

	cs.patchHelper, err = patch.NewHelper(params.K8sMachine, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	return &MachineScope{
		ClusterScope: *cs,
		Machine:      params.Machine,
		K8sMachine:   params.K8sMachine,
		Namespace:    params.Namespace,
	}, nil
}

// MachineScope defines the basic context for an actuator to operate upon.
type MachineScope struct {
	ClusterScope
	Machine    *clusterv1.Machine
	K8sMachine *infrav1.K8sMachine
	Namespace  string
}

// SetReady sets the ready field on the machine.
func (m *MachineScope) SetReady(ready bool) {
	m.K8sMachine.Status.Ready = ready
}

// Close closes the current scope persisting the cluster configuration and status.
func (m *MachineScope) Close(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.K8sMachine)
}

// PatchObject persists the machine spec and status.
func (m *MachineScope) PatchObject(ctx context.Context) error {
	return m.patchHelper.Patch(ctx, m.K8sMachine)
}
