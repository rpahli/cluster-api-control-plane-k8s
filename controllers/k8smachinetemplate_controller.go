package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	controlplanev1 "sigs.k8s.io/cluster-api-provider-nested/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/scope"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/services/machinetemplate"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// K8sMachineTemplateReconciler reconciles a K8sMachineTemplate object.
type K8sMachineTemplateReconciler struct {
	client.Client
	Log              logr.Logger
	Scheme           *runtime.Scheme
	WatchFilterValue string
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8smachinetemplate,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8smachinetemplate/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8smachinetemplate/finalizers,verbs=update

func (r *K8sMachineTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	machineTemplate := &controlplanev1.K8sMachineTemplate{}
	if err := r.Get(ctx, req.NamespacedName, machineTemplate); err != nil {
		log.Error(err, "unable to fetch K8sMachineTemplate")
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	log = log.WithValues("K8sMachineTemplate", klog.KObj(machineTemplate))

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, machineTemplate.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster owner reference or cluster does not exist")
		return reconcile.Result{}, fmt.Errorf("failed to get owner cluster: %w", err)
	}

	log = log.WithValues("Cluster", klog.KObj(cluster))

	// Requeue if cluster has no infrastructure yet.
	if cluster == nil || cluster.Spec.InfrastructureRef == nil {
		return reconcile.Result{Requeue: true}, nil
	}

	machineTemplateScope, err := scope.NewK8sMachineTemplateScope(ctx, scope.K8sMachineTemplateScopeParams{
		Client:             r.Client,
		Logger:             &log,
		K8sMachineTemplate: machineTemplate,
	})
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to create scope: %w", err)
	}

	// Always close the scope when exiting this function so we can persist any HCloudMachine changes.
	defer func() {
		if err := machineTemplateScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// check whether rate limit has been reached and if so, then wait.
	if wait := reconcileRateLimit(machineTemplate); wait {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if !machineTemplate.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineTemplateScope)
	}

	return r.reconcileNormal(ctx, machineTemplateScope)
}

func (r *K8sMachineTemplateReconciler) reconcileDelete(ctx context.Context, machineTemplateScope *scope.K8sMachineTemplateScope) (reconcile.Result, error) {
	controllerutil.RemoveFinalizer(machineTemplateScope.K8sMachineTemplate, controlplanev1.MachineFinalizer)
	return reconcile.Result{}, nil
}

func (r *K8sMachineTemplateReconciler) reconcileNormal(ctx context.Context, machineTemplateScope *scope.K8sMachineTemplateScope) (reconcile.Result, error) {
	hcloudMachineTemplate := machineTemplateScope.K8sMachineTemplate

	// If the HCloudMachineTemplate doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(machineTemplateScope.K8sMachineTemplate, controlplanev1.MachineFinalizer)

	// Register the finalizer immediately to avoid orphaning HCloud resources on delete
	if err := machineTemplateScope.PatchObject(ctx); err != nil {
		return reconcile.Result{}, err
	}

	// reconcileNormal machine template
	if err := machinetemplate.NewService(machineTemplateScope).Reconcile(ctx); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to reconcileNormal machine template for K8sMachineTemplate %s/%s: %w",
			hcloudMachineTemplate.Namespace, hcloudMachineTemplate.Name, err)
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sMachineTemplateReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&controlplanev1.K8sMachineTemplate{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(ctrl.LoggerFrom(ctx), r.WatchFilterValue)).
		Complete(r)
}
