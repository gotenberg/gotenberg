package gotenberg

import (
	"context"
	"log/slog"
	"os"
)

// ModuleMock is a mock for the [Module] interface.
type ModuleMock struct {
	DescriptorMock func() ModuleDescriptor
}

// Descriptor calls DescriptorMock.
func (mod *ModuleMock) Descriptor() ModuleDescriptor {
	return mod.DescriptorMock()
}

// ProvisionerMock is a mock for the [Provisioner] interface.
type ProvisionerMock struct {
	ProvisionMock func(*Context) error
}

// Provision calls ProvisionMock.
func (mod *ProvisionerMock) Provision(ctx *Context) error {
	return mod.ProvisionMock(ctx)
}

// ValidatorMock is a mock for the [Validator] interface.
type ValidatorMock struct {
	ValidateMock func() error
}

// Validate calls ValidateMock.
func (mod *ValidatorMock) Validate() error {
	return mod.ValidateMock()
}

// DebuggableMock is a mock for the [Debuggable] interface.
type DebuggableMock struct {
	DebugMock func() map[string]any
}

// Debug calls DebugMock.
func (mod *DebuggableMock) Debug() map[string]any {
	return mod.DebugMock()
}

// ProcessMock is a mock for the [Process] interface.
type ProcessMock struct {
	StartMock   func(ctx context.Context, logger *slog.Logger) error
	StopMock    func(logger *slog.Logger) error
	HealthyMock func(logger *slog.Logger) bool
}

// Start calls StartMock.
func (p *ProcessMock) Start(ctx context.Context, logger *slog.Logger) error {
	return p.StartMock(ctx, logger)
}

// Stop calls StopMock.
func (p *ProcessMock) Stop(logger *slog.Logger) error {
	return p.StopMock(logger)
}

// Healthy calls HealthyMock.
func (p *ProcessMock) Healthy(logger *slog.Logger) bool {
	return p.HealthyMock(logger)
}

// ProcessSupervisorMock is a mock for the [ProcessSupervisor] interface.
type ProcessSupervisorMock struct {
	LaunchMock   func(ctx context.Context) error
	ShutdownMock func() error
	HealthyMock  func() bool
	RunMock      func(ctx context.Context, logger *slog.Logger, task func() error) error
}

// Launch calls LaunchMock.
func (s *ProcessSupervisorMock) Launch(ctx context.Context) error {
	return s.LaunchMock(ctx)
}

// Shutdown calls ShutdownMock.
func (s *ProcessSupervisorMock) Shutdown() error {
	return s.ShutdownMock()
}

// Healthy calls HealthyMock.
func (s *ProcessSupervisorMock) Healthy() bool {
	return s.HealthyMock()
}

// Run calls RunMock.
func (s *ProcessSupervisorMock) Run(ctx context.Context, logger *slog.Logger, task func() error) error {
	return s.RunMock(ctx, logger, task)
}

// MkdirAllMock is a mock for the [MkdirAll] interface.
type MkdirAllMock struct {
	MkdirAllMock func(path string, perm os.FileMode) error
}

// MkdirAll calls MkdirAllMock.
func (mkdirAll *MkdirAllMock) MkdirAll(path string, perm os.FileMode) error {
	return mkdirAll.MkdirAllMock(path, perm)
}

// PathRenameMock is a mock for the [PathRename] interface.
type PathRenameMock struct {
	RenameMock func(oldpath, newpath string) error
}

// Rename calls RenameMock.
func (rename *PathRenameMock) Rename(oldpath, newpath string) error {
	return rename.RenameMock(oldpath, newpath)
}

// Interface guards.
var (
	_ Module            = (*ModuleMock)(nil)
	_ Validator         = (*ValidatorMock)(nil)
	_ Process           = (*ProcessMock)(nil)
	_ ProcessSupervisor = (*ProcessSupervisorMock)(nil)
	_ MkdirAll          = (*MkdirAllMock)(nil)
	_ PathRename        = (*PathRenameMock)(nil)
)
