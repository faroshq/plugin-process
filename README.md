# Plugin Process

Plugin to manage system process on the server.

Current implementation support only `Systemd` process manager.

## Development

Folder structure:
`pkg/server` - contains server code
`pkg/agent` - contains agent code
`pkg/apis` - contains plugin api code
`cmd` - entrypoints for server and agent
