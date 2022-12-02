package plugin

import (
	"context"
	"embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/phayes/freeport"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/faroshq/faros-hub/pkg/plugins"
	"github.com/faroshq/plugin-services/pkg/agent/systemd"
	servicesv1alpha1 "github.com/faroshq/plugin-services/pkg/apis/services/v1alpha1"
	utiltemplate "github.com/faroshq/plugin-services/pkg/util/template"
	"github.com/faroshq/plugin-services/pkg/util/version"
)

var (
	scheme = runtime.NewScheme()
	// pluginName is the name of the plugin. Should match apiresourceschemas name.
	pluginName = "systemds.services.plugins.faros.sh"
)

func init() {
	utilruntime.Must(servicesv1alpha1.AddToScheme(scheme))
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
	return pluginName, nil
}

func (s *SystemD) GetVersion(context.Context) (string, error) {
	return version.Get().Version, nil
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

//go:embed data/*
var content embed.FS

func (s *SystemD) GetAPIResourceSchema(ctx context.Context) ([]byte, error) {
	entries, err := content.ReadDir("data")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "apiresourceschemas.yaml") {
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
	var apiExportName, apiResourceSchemaName string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "apiexport.yaml.template") {
			apiExportName = entry.Name()
		}
		if strings.Contains(entry.Name(), "apiresourceschemas.yaml") {
			apiResourceSchemaName = entry.Name()
		}
	}
	if apiExportName == "" || apiResourceSchemaName == "" {
		return nil, fmt.Errorf("apiexport or apiresourceschema not found")
	}

	// get name for apiresourceschemas
	data, err := content.ReadFile("data/" + apiResourceSchemaName)
	if err != nil {
		return nil, fmt.Errorf("failed to read apiresourceschema: %w", err)
	}

	var unstructured unstructured.Unstructured
	err = yaml.Unmarshal(data, &unstructured)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal apiresourceschema: %w", err)
	}

	data, err = content.ReadFile("data/" + apiExportName)
	if err != nil {
		return nil, fmt.Errorf("failed to read apiexport: %w", err)
	}

	args := utiltemplate.TemplateArgs{
		Name:                 fmt.Sprintf("%s.%s", version.Get().Version, pluginName),
		LatestResourceSchema: unstructured.GetName(),
	}
	apiExportBytes, err := utiltemplate.RenderTemplate(data, args)
	if err != nil {
		return nil, fmt.Errorf("failed to render apiexport: %w", err)
	}

	return apiExportBytes, nil

}
