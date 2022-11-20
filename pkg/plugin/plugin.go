package plugin

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"

	pluginsv1alpha1 "github.com/faroshq/plugin-process/pkg/apis/plugins/v1alpha1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(pluginsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
}

// Interface is the interface that plugins must implement.
// TODO: move to faros-hub repo
type Interface interface {
	// Name returns the name of the plugin.
	Name() string
	// Init initializes the plugin.
	Init(ctx context.Context, name, namespace string, config *rest.Config) error
	// Run runs the plugin.
	Run(ctx context.Context) error
	// Stop stops the plugin.
	Stop() error
}
