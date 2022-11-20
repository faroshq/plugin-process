package systemd

import (
	"context"

	pluginsv1alpha1 "github.com/faroshq/plugin-process/pkg/apis/plugins/v1alpha1"
	"github.com/go-logr/logr"
	conditionsv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
	"github.com/kcp-dev/kcp/pkg/apis/third_party/conditions/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) createOrUpdate(ctx context.Context, logger logr.Logger, systemd *pluginsv1alpha1.Systemd) (ctrl.Result, error) {
	patch := client.MergeFrom(systemd.DeepCopy())
	conditions.MarkTrue(systemd, conditionsv1alpha1.ReadyCondition)

	if err := r.Status().Patch(ctx, systemd, patch); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}
