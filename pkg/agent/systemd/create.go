package systemd

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/davecgh/go-spew/spew"
	"github.com/faroshq/plugin-services/pkg/apis/services/v1alpha1"
	servicesv1alpha1 "github.com/faroshq/plugin-services/pkg/apis/services/v1alpha1"
	"github.com/go-logr/logr"
	conditionsv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/third_party/conditions/apis/conditions/v1alpha1"
	"github.com/kcp-dev/kcp/pkg/apis/third_party/conditions/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultActivationMode = servicesv1alpha1.ActivationModeReplace
	defaultEnableMode     = servicesv1alpha1.EnableModeRuntimeOnly
)

func (r *Reconciler) createOrUpdate(ctx context.Context, logger logr.Logger, systemd *servicesv1alpha1.Systemd) (ctrl.Result, error) {
	patch := client.MergeFrom(systemd.DeepCopy())
	conditions.MarkTrue(systemd, conditionsv1alpha1.ReadyCondition)

	systemd.Status.Units = make([]servicesv1alpha1.UnitStatus, len(systemd.Spec.Units))

	conn, err := dbus.NewWithContext(ctx)
	if err != nil {
		logger.Error(err, "failed to connect to systemd")
		conditions.MarkFalse(systemd, conditionsv1alpha1.ReadyCondition, "FailedToConnect", "Failed to connect to systemd", err.Error())
		return ctrl.Result{
			Requeue: true,
		}, err
	}
	defer conn.Close()

	for _, unit := range systemd.Spec.Units {
		status, err := r.handleUnit(ctx, logger, conn, unit)
		if err != nil {
			logger.Error(err, "failed to handle unit", "unit", spew.Sdump(unit))
			conditions.MarkFalse(systemd, conditionsv1alpha1.ReadyCondition, "FailedToHandleUnit", "Failed to handle unit", err.Error())
			return ctrl.Result{
				Requeue: true,
			}, err
		}
		systemd.Status.Units = append(systemd.Status.Units, v1alpha1.UnitStatus{
			Name:          unit.Name,
			Status:        status.Status,
			Error:         status.Error.Error(),
			DesiredStatus: unit.DesiredStatus.String(),
		})
	}

	if err := r.Status().Patch(ctx, systemd, patch); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

type status struct {
	Name   string
	Status string
	Error  error
}

// handleUnit handles a single unit. It returns error if overall operation failed.
// It will return individual service status in status object and it should be handled by caller.
func (r *Reconciler) handleUnit(ctx context.Context, logger logr.Logger, conn *dbus.Conn, unit servicesv1alpha1.Unit) (*status, error) {
	u := unit.DeepCopy()
	if u.ActivationMode == "" {
		u.ActivationMode = defaultActivationMode
	}

	s := &status{
		Name: u.Name,
	}

	persistent := u.EnableMode == servicesv1alpha1.EnableModePersistent
	if u.EnableMode == "" {
		u.EnableMode = defaultEnableMode
	}

	switch unit.DesiredStatus {
	case servicesv1alpha1.ServiceStatusEnabled:
		_, _, err := conn.EnableUnitFilesContext(ctx, []string{unit.Name}, persistent, false)
		if err != nil {
			s.Error = err
		}
	case servicesv1alpha1.ServiceStatusDisabled:
		_, err := conn.DisableUnitFilesContext(ctx, []string{unit.Name}, persistent)
		if err != nil {
			s.Error = err
		}
	case servicesv1alpha1.ServiceStatusStarted:
		reschan := make(chan string)
		_, err := conn.StartUnitContext(ctx, unit.Name, u.ActivationMode.String(), reschan)
		if err != nil {
			s.Error = err
		} else {
			// wait for done
			job := <-reschan
			if job != "done" {
				s.Error = fmt.Errorf("job != done with status: %s", job)
			}
		}
	case servicesv1alpha1.ServiceStatusStopped:
		reschan := make(chan string)
		_, err := conn.StopUnitContext(ctx, unit.Name, u.ActivationMode.String(), reschan)
		if err != nil {
			s.Error = err
		} else {
			// wait for done
			<-reschan
		}
	case servicesv1alpha1.ServiceStatusEnabledAndStarted:
		_, _, err := conn.EnableUnitFilesContext(ctx, []string{unit.Name}, persistent, false)
		if err != nil {
			s.Error = err
		} else {
			reschan := make(chan string)
			_, err := conn.StartUnitContext(ctx, unit.Name, u.ActivationMode.String(), reschan)
			if err != nil {
				s.Error = err
			} else {
				// wait for done
				job := <-reschan
				if job != "done" {
					s.Error = fmt.Errorf("job != done with status: %s", job)
				}
			}
		}
	case servicesv1alpha1.ServiceStatusDisabledAndStopped:
		_, err := conn.DisableUnitFilesContext(ctx, []string{unit.Name}, persistent)
		if err != nil {
			s.Error = err
		} else {
			reschan := make(chan string)
			_, err := conn.StopUnitContext(ctx, unit.Name, u.ActivationMode.String(), reschan)
			if err != nil {
				s.Error = err
			} else {
				// wait for done
				<-reschan
			}
		}
	}

	// check status
	units, err := conn.ListJobsContext(ctx)
	if err != nil {
		return nil, err
	}

	for _, u := range units {
		if u.Unit == unit.Name {
			s.Status = u.Status
			break
		}
	}

	return s, nil
}
