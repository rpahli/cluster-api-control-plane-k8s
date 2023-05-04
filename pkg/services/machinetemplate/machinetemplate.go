package machinetemplate

import (
	"context"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/scope"
)

// Service defines struct with HCloudMachineTemplate scope to reconcile HCloud machine templates.
type Service struct {
	scope *scope.K8sMachineTemplateScope
}

// NewService outs a new service with HCloudMachineTemplate scope.
func NewService(scope *scope.K8sMachineTemplateScope) *Service {
	return &Service{
		scope: scope,
	}
}

// Reconcile implements reconcilement of HCloud machines.
func (s *Service) Reconcile(ctx context.Context) error {
	/*capacity, err := s.getCapacity(ctx)
	if err != nil {
		return fmt.Errorf("failed to get capacity: %w", err)
	}

	s.scope.K8sMachineTemplate.Status.Capacity = capacity*/
	return nil
}
