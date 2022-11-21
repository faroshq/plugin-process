package plugin

import (
	"context"
	"embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/phayes/freeport"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/faroshq/faros-hub/pkg/plugins"
	"github.com/faroshq/plugin-process/pkg/agent/systemd"
	pluginsv1alpha1 "github.com/faroshq/plugin-process/pkg/apis/plugins/v1alpha1"
	utiltemplate "github.com/faroshq/plugin-process/pkg/util/template"
	"github.com/faroshq/plugin-process/pkg/util/version"
)

var (
	scheme     = runtime.NewScheme()
	pluginName = "systemd.plugins.faros.sh"
)

func init() {
	utilruntime.Must(pluginsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
}

var _ plugins.Interface = &SystemD{}

type SystemD struct {
	name      string
	namespace string
	client    client.Client
	schema    *runtime.Scheme
	manager   manager.Manager
}

func (s *SystemD) GetName(context.Context) (string, error) {
	return "process.systemd", nil
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
		klog.Error(err, "unable to create controller", pluginName)
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

func (s *SystemD) Stop(ctx context.Context) error {
	return nil
}

//go:embed data/*.yaml
var content embed.FS

func (s *SystemD) GetAPIResourceSchema(ctx context.Context) ([]byte, error) {
	entries, err := content.ReadDir("data")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "apiresourceschemas") {
			return content.ReadFile("data/" + entry.Name())
		}
	}
	return nil, fmt.Errorf("apiresourceschemas not found")
}

func (s *SystemD) GetAPIExportSchema(ctx context.Context) ([]byte, error) {
	entries, err := content.ReadDir("data")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "apiexport") {

			data, err := content.ReadFile("data/" + entry.Name())
			if err != nil {
				return nil, fmt.Errorf("failed to read apiexport: %w", err)
			}
			args := utiltemplate.TemplateArgs{
				Name:                 fmt.Sprintf("%s.%s", version.Get().Version, pluginName),
				LatestResourceSchema: strings.TrimSuffix(entry.Name(), ".yaml"),
			}
			apiExportBytes, err := utiltemplate.RenderTemplate(data, args)
			if err != nil {
				return nil, fmt.Errorf("failed to render apiexport: %w", err)
			}

			return apiExportBytes, nil
		}
	}
	return nil, fmt.Errorf("apiexport not found")
}
