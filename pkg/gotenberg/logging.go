package gotenberg

import "go.uber.org/zap"

// LoggerProvider is an interface for a module that supplies a method for
// creating a [zap.Logger] instance for use by other modules.
//
//	func (m *YourModule) Provision(ctx *gotenberg.Context) error {
//		provider, _ := ctx.Module(new(gotenberg.LoggerProvider))
//		logger, _   := provider.(gotenberg.LoggerProvider).Logger(m)
//	}
type LoggerProvider interface {
	Logger(mod Module) (*zap.Logger, error)
}
