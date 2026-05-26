package handler

// Handlers contains all HTTP handlers
type Handlers struct {
	Gateway       *GatewayHandler
	OpenAIGateway *OpenAIGatewayHandler
}

// BuildInfo contains build-time information
type BuildInfo struct {
	Version   string
	BuildType string // "source" for manual builds, "release" for CI builds
}
