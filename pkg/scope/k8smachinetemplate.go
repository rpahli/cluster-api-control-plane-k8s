package scope

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/klog/v2/klogr"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8sMachineTemplateScopeParams defines the input parameters used to create a new scope.
type K8sMachineTemplateScopeParams struct {
	Client             client.Client
	Logger             *logr.Logger
	K8sMachineTemplate *infrav1.K8sMachineTemplate
}

// NewK8sMachineTemplateScope creates a new Scope from the supplied parameters.
// This is meant to be called for each reconcile iteration.
func NewK8sMachineTemplateScope(ctx context.Context, params K8sMachineTemplateScopeParams) (*K8sMachineTemplateScope, error) {

	if params.Logger == nil {
		logger := klogr.New()
		params.Logger = &logger
	}

	helper, err := patch.NewHelper(params.K8sMachineTemplate, params.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to init patch helper: %w", err)
	}

	return &K8sMachineTemplateScope{
		Logger:             params.Logger,
		Client:             params.Client,
		K8sMachineTemplate: params.K8sMachineTemplate,
		patchHelper:        helper,
	}, nil
}

type K8sMachineTemplateScope struct {
	*logr.Logger
	Client             client.Client
	patchHelper        *patch.Helper
	K8sMachineTemplate *infrav1.K8sMachineTemplate
}

// Name returns the HCloudMachineTemplate name.
func (s *K8sMachineTemplateScope) Name() string {
	return s.K8sMachineTemplate.Name
}

// Namespace returns the namespace name.
func (s *K8sMachineTemplateScope) Namespace() string {
	return s.K8sMachineTemplate.Namespace
}

// Close closes the current scope persisting the cluster configuration and status.
func (s *K8sMachineTemplateScope) Close(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.K8sMachineTemplate)
}

// PatchObject persists the machine spec and status.
func (s *K8sMachineTemplateScope) PatchObject(ctx context.Context) error {
	return s.patchHelper.Patch(ctx, s.K8sMachineTemplate)
}
