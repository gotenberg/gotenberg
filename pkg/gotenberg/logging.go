package gotenberg

import "go.uber.org/zap"

// LoggerProvider is a module interface which exposes a method for creating a
// zap.Logger for other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(gotenberg.LoggerProvider))
//		logger, _   := provider.(gotenberg.LoggerProvider).Logger(m)
//	}
type LoggerProvider interface {
	Logger(mod Module) (*zap.Logger, error)
}
