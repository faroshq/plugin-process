package main

import (
	"github.com/faroshq/faros-hub/pkg/plugins/shared"
	farosplugin "github.com/faroshq/plugin-process/pkg/plugin"
	"github.com/hashicorp/go-plugin"
)

func main() {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: shared.Handshake,
		Plugins: map[string]plugin.Plugin{
			"plugin": &shared.DevicePlugin{Impl: &farosplugin.SystemD{}},
		},

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
