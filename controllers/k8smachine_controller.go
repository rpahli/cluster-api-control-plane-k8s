package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	infrav1 "sigs.k8s.io/cluster-api-provider-nested/api/infrastructure/v1beta1"
	"sigs.k8s.io/cluster-api-provider-nested/pkg/scope"
	server "sigs.k8s.io/cluster-api-provider-nested/pkg/services"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

// K8sMachineReconciler reconciles a K8sMachine object.
type K8sMachineReconciler struct {
	client.Client
	APIReader        client.Reader
	Log              logr.Logger
	Scheme           *runtime.Scheme
	WatchFilterValue string
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8smachine,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8smachine/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=k8smachine/finalizers,verbs=update

func (r *K8sMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := r.Log.WithValues("k8smachine", req.NamespacedName)
	log.Info("Reconciling K8sMachine ...")

	// Fetch the HCloudMachine instance.
	k8sMachine := &infrav1.K8sMachine{}
	err := r.Get(ctx, req.NamespacedName, k8sMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	log = log.WithValues("K8sMachine", klog.KObj(k8sMachine))

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, k8sMachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("Machine", klog.KObj(machine))

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return reconcile.Result{}, nil
	}

	if annotations.IsPaused(cluster, k8sMachine) {
		log.Info("K8sMachine or linked Cluster is marked as paused. Won't reconcile")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("Cluster", klog.KObj(cluster))

	k8sCluster := &infrav1.K8sCluster{}

	k8sClusterName := client.ObjectKey{
		Namespace: k8sMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Client.Get(ctx, k8sClusterName, k8sCluster); err != nil {
		log.Info("K8sCluster is not available yet")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("K8sCluster", klog.KObj(k8sCluster))
	ctx = ctrl.LoggerInto(ctx, log)

	// Create the scope.
	// secretManager := secretutil.NewSecretManager(log, r.ManagerClient, r.APIReader)
	/*	hcloudToken, hetznerSecret, err := getAndValidateHCloudToken(ctx, req.Namespace, k8sCluster, secretManager)
		if err != nil {
			return hcloudTokenErrorResult(ctx, err, k8sMachine, infrav1.InstanceReadyCondition, r.ManagerClient)
		}*/

	// namespace := "default"

	// secretManager.FindSecret(ctx, types.NamespacedName{Name: "", Namespace: })

	machineScope, err := scope.NewMachineScope(ctx, scope.MachineScopeParams{
		ClusterScopeParams: scope.ClusterScopeParams{
			Client:     r.Client,
			Logger:     log,
			Cluster:    cluster,
			K8sCluster: k8sCluster,
			APIReader:  r.APIReader,
		},
		Machine:    machine,
		K8sMachine: k8sMachine,
		Namespace:  "default",
	})

	// Always close the scope when exiting this function so we can persist any HCloudMachine changes.
	defer func() {
		if err := machineScope.Close(ctx); err != nil && reterr == nil {
			reterr = err
		}
	}()

	// Check whether rate limit has been reached and if so, then wait.
	if wait := reconcileRateLimit(k8sMachine); wait {
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if !k8sMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, machineScope)
	}

	return r.reconcileNormal(ctx, machineScope)
}

func (r *K8sMachineReconciler) reconcileDelete(ctx context.Context, machineScope *scope.MachineScope) (reconcile.Result, error) {
	hcloudMachine := machineScope.K8sMachine

	// Delete servers.
	result, err := server.NewService(machineScope).Delete(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to delete servers for HCloudMachine %s/%s: %w", hcloudMachine.Namespace, hcloudMachine.Name, err)
	}
	emptyResult := reconcile.Result{}
	if result != emptyResult {
		return result, nil
	}
	// Machine is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(machineScope.K8sMachine, infrav1.MachineFinalizer)

	return reconcile.Result{}, nil
}

func (r *K8sMachineReconciler) reconcileNormal(ctx context.Context, machineScope *scope.MachineScope) (reconcile.Result, error) {
	hcloudMachine := machineScope.K8sMachine

	// If the HCloudMachine doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(machineScope.K8sMachine, infrav1.MachineFinalizer)

	// Register the finalizer immediately to avoid orphaning HCloud resources on delete.
	if err := machineScope.PatchObject(ctx); err != nil {
		return reconcile.Result{}, err
	}

	/*	_, err := kubeadm.GenerateTemplates(r.Log, machineScope.Cluster.Name)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to reconcile server for HCloudMachine %s/%s: %w",
				hcloudMachine.Namespace, hcloudMachine.Name, err)
		}*/
	// reconcile server
	result, err := server.NewService(machineScope).Reconcile(ctx)
	if err != nil {
		return result, fmt.Errorf("failed to reconcile server for HCloudMachine %s/%s: %w",
			hcloudMachine.Namespace, hcloudMachine.Name, err)
	}
	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8sMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)
	c, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.K8sMachine{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("K8sMachine"))),
		).
		Watches(
			&source.Kind{Type: &infrav1.K8sCluster{}},
			handler.EnqueueRequestsFromMapFunc(r.K8sClusterToHCloudMachines(ctx)),
		).
		Build(r)
	if err != nil {
		return fmt.Errorf("error creating controller: %w", err)
	}

	clusterToObjectFunc, err := util.ClusterToObjectsMapper(r.Client, &infrav1.K8sMachineList{}, mgr.GetScheme())
	if err != nil {
		return fmt.Errorf("failed to create mapper for Cluster to K8sMachines: %w", err)
	}

	// Add a watch on clusterv1.Cluster object for unpause & ready notifications.
	if err := c.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
		predicates.ClusterUnpausedAndInfrastructureReady(log),
	); err != nil {
		return fmt.Errorf("failed adding a watch for ready clusters: %w", err)
	}
	return nil
}

// K8sClusterToHCloudMachines is a handler.ToRequestsFunc to be used to enqeue requests for reconciliation
// of HCloudMachines.
func (r *K8sMachineReconciler) K8sClusterToHCloudMachines(ctx context.Context) handler.MapFunc {
	return func(o client.Object) []reconcile.Request {
		result := []reconcile.Request{}

		log := log.FromContext(ctx)

		c, ok := o.(*infrav1.K8sCluster)
		if !ok {
			log.Error(fmt.Errorf("expected a K8sCluster but got a %T", o), "failed to get K8sMachine for K8sCluster")
			return nil
		}

		log = log.WithValues("objectMapper", "k8sClusterToHCloudMachines", "namespace", c.Namespace, "k8sCluster", c.Name)

		// Don't handle deleted K8sCluster
		if !c.ObjectMeta.DeletionTimestamp.IsZero() {
			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, c.ObjectMeta)
		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			return result
		case err != nil:
			return result
		}

		labels := map[string]string{clusterv1.ClusterNameLabel: cluster.Name}
		machineList := &clusterv1.MachineList{}
		if err := r.List(ctx, machineList, client.InNamespace(c.Namespace), client.MatchingLabels(labels)); err != nil {
			log.Error(err, "failed to list Machines, skipping mapping")
			return nil
		}
		for _, m := range machineList.Items {
			log = log.WithValues("machine", m.Name)
			if m.Spec.InfrastructureRef.GroupVersionKind().Kind != "K8sMachine" {
				continue
			}
			if m.Spec.InfrastructureRef.Name == "" {
				continue
			}
			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}

			result = append(result, reconcile.Request{NamespacedName: name})
		}

		return result
	}
}
