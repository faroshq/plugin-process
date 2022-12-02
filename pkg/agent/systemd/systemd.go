package systemd

import (
	"context"

	conditionsv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
	"github.com/kcp-dev/kcp/pkg/apis/third_party/conditions/util/conditions"
	"github.com/kcp-dev/logicalcluster/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	servicesv1alpha1 "github.com/faroshq/plugin-services/pkg/apis/services/v1alpha1"
)

// Reconciler reconciles a SystemD object
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=services.plugins.faros.sh,resources=systemd,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=services.plugins.faros.sh,resources=systemd/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=services.plugins.faros.sh,resources=systemd/finalizers,verbs=update

// Reconcile reconciles a SystemD object
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Include the clusterName from req.ObjectKey in the logger, similar to the namespace and name keys that are already
	// there.
	logger = logger.WithValues("clusterName", req.ClusterName).WithValues("namespace", req.Namespace).WithValues("name", req.Name)

	// Add the logical cluster to the context
	ctx = logicalcluster.WithCluster(ctx, logicalcluster.New(req.ClusterName))

	var systemd servicesv1alpha1.Systemd
	if err := r.Get(ctx, req.NamespacedName, &systemd); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	var result ctrl.Result
	var err error
	if systemd.DeletionTimestamp.IsZero() {
		result, err = r.createOrUpdate(ctx, logger, systemd.DeepCopy())
	}
	if err != nil {
		systemdCopy := systemd.DeepCopy()
		conditions.MarkFalse(
			systemdCopy,
			conditionsv1alpha1.ReadyCondition,
			err.Error(),
			conditionsv1alpha1.ConditionSeverityError,
			"Error configuring Registration: %v",
			err,
		)
		if err := r.Status().Patch(ctx, systemdCopy, client.MergeFrom(&systemd)); err != nil {
			return result, err
		}
	}
	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// TODO: scope to a specific agent
		For(&servicesv1alpha1.Systemd{}).
		Complete(r)
}
