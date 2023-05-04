package server

import (
	"context"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/scope"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Service defines struct with machine scope to reconcile HCloudMachines.
type Service struct {
	scope *scope.MachineScope
}

// NewService outs a new service with machine scope.
func NewService(scope *scope.MachineScope) *Service {
	return &Service{
		scope: scope,
	}
}

// Reconcile implements reconcilement of HCloudMachines.
func (s *Service) Reconcile(ctx context.Context) (res reconcile.Result, err error) {
	return res, nil
}

// Delete implements delete method of server.
func (s *Service) Delete(ctx context.Context) (res reconcile.Result, err error) {
	return res, nil
}
