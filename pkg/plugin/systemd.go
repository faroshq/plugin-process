package plugin

import (
	"context"
	"fmt"
	"strconv"

	"github.com/phayes/freeport"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/faroshq/plugin-process/pkg/agent/systemd"
	pluginsv1alpha1 "github.com/faroshq/plugin-process/pkg/apis/plugins/v1alpha1"
)

var _ Interface = &SystemD{}

type SystemD struct {
	name      string
	namespace string
	client    client.Client
	schema    *runtime.Scheme
	manager   manager.Manager
}

func (s *SystemD) Name() string {
	return "process.systemd"
}

func (s *SystemD) Init(ctx context.Context, name, namespace string, config *rest.Config) error {
	port, err := freeport.GetFreePort()
	if err != nil {
		return err
	}

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     ":" + strconv.Itoa(port),
		Port:                   port,
		HealthProbeBindAddress: ":" + strconv.Itoa(port),
		LeaderElection:         false,
	}

	mgr, err := ctrl.NewManager(config, options)
	if err != nil {
		return err
	}

	s.client = mgr.GetClient()
	s.schema = mgr.GetScheme()
	s.name = name
	s.namespace = namespace

	if err = (&systemd.Reconciler{
		Client: s.client,
		Scheme: s.schema,
	}).SetupWithManager(mgr); err != nil {
		klog.Error(err, "unable to create controller", "systemd.plugins.faros.sh")
		return err
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		klog.Error(err, "unable to set up ready check")
		return err
	}

	s.manager = mgr

	return nil
}

func (s *SystemD) Run(ctx context.Context) error {
	return s.manager.Start(ctx)
}

func (s *SystemD) Stop() error {
	return nil
}

// bootstrap will new instance of plugin for it to be reporting back to hub
func (s *SystemD) bootstrap(ctx context.Context) error {
	obj := pluginsv1alpha1.Systemd{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: s.namespace,
		},
		Spec: pluginsv1alpha1.SystemdSpec{},
	}

	err := s.client.Get(ctx, client.ObjectKey{
		Namespace: obj.Namespace,
		Name:      obj.Name,
	}, &obj)
	switch {
	case apierrors.IsNotFound(err):
		err := s.client.Create(ctx, &obj, &client.CreateOptions{})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create plugin object: %s", err)
		}
	case err == nil:
		obj.ResourceVersion = ""
		// TODO: update path

		err = s.client.Patch(ctx, &obj, client.MergeFrom(obj.DeepCopy()), &client.PatchOptions{})
		if err != nil {
			return fmt.Errorf("failed to patch plugin object %s", err)
		}
	default:
		return fmt.Errorf("failed to get the plugin object %s", err)
	}

	return nil
}
