package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/google/wire"
)

// ProvideHandlers creates the Handlers struct
func ProvideHandlers(
	gatewayHandler *GatewayHandler,
	openaiGatewayHandler *OpenAIGatewayHandler,
	_ *service.IdempotencyCoordinator,
	_ *service.IdempotencyCleanupService,
) *Handlers {
	return &Handlers{
		Gateway:       gatewayHandler,
		OpenAIGateway: openaiGatewayHandler,
	}
}

// ProviderSet is the Wire provider set for all handlers
var ProviderSet = wire.NewSet(
	NewGatewayHandler,
	NewOpenAIGatewayHandler,
	ProvideHandlers,
)
