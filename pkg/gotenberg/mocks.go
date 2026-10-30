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

func (mod *ModuleMock) Descriptor() ModuleDescriptor {
	return mod.DescriptorMock()
}

// ProvisionerMock is a mock for the [Provisioner] interface.
type ProvisionerMock struct {
	ProvisionMock func(*Context) error
}

func (mod *ProvisionerMock) Provision(ctx *Context) error {
	return mod.ProvisionMock(ctx)
}

// ValidatorMock is a mock for the [Validator] interface.
type ValidatorMock struct {
	ValidateMock func() error
}

func (mod *ValidatorMock) Validate() error {
	return mod.ValidateMock()
}

type DebuggableMock struct {
	DebugMock func() map[string]any
}

func (mod *DebuggableMock) Debug() map[string]any {
	return mod.DebugMock()
}

// ProcessMock is a mock for the [Process] interface.
type ProcessMock struct {
	StartMock   func(logger *slog.Logger) error
	StopMock    func(logger *slog.Logger) error
	HealthyMock func(logger *slog.Logger) bool
}

func (p *ProcessMock) Start(logger *slog.Logger) error {
	return p.StartMock(logger)
}

func (p *ProcessMock) Stop(logger *slog.Logger) error {
	return p.StopMock(logger)
}

func (p *ProcessMock) Healthy(logger *slog.Logger) bool {
	return p.HealthyMock(logger)
}

// ProcessSupervisorMock is a mock for the [ProcessSupervisor] interface.
type ProcessSupervisorMock struct {
	LaunchMock                  func() error
	ShutdownMock                func() error
	HealthyMock                 func() bool
	RunMock                     func(ctx context.Context, logger *slog.Logger, task func() error) error
	ReqQueueSizeMock            func() int64
	RestartsCountMock           func() int64
	ActiveTasksCountMock        func() int64
	ConversionsSinceRestartMock func() int64
}

func (s *ProcessSupervisorMock) Launch() error {
	return s.LaunchMock()
}

func (s *ProcessSupervisorMock) Shutdown() error {
	return s.ShutdownMock()
}

func (s *ProcessSupervisorMock) Healthy() bool {
	return s.HealthyMock()
}

func (s *ProcessSupervisorMock) Run(ctx context.Context, logger *slog.Logger, task func() error) error {
	return s.RunMock(ctx, logger, task)
}

func (s *ProcessSupervisorMock) ReqQueueSize() int64 {
	return s.ReqQueueSizeMock()
}

func (s *ProcessSupervisorMock) RestartsCount() int64 {
	return s.RestartsCountMock()
}

func (s *ProcessSupervisorMock) ActiveTasksCount() int64 {
	return s.ActiveTasksCountMock()
}

func (s *ProcessSupervisorMock) ConversionsSinceRestart() int64 {
	return s.ConversionsSinceRestartMock()
}

// MkdirAllMock is a mock for the [MkdirAll] interface.
type MkdirAllMock struct {
	MkdirAllMock func(path string, perm os.FileMode) error
}

func (mkdirAll *MkdirAllMock) MkdirAll(path string, perm os.FileMode) error {
	return mkdirAll.MkdirAllMock(path, perm)
}

// PathRenameMock is a mock for the [PathRename] interface.
type PathRenameMock struct {
	RenameMock func(oldpath, newpath string) error
}

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
