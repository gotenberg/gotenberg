package gotenberg

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCommand(t *testing.T) {
	cmd := Command(zap.NewNop(), "foo")

	if !cmd.process.SysProcAttr.Setpgid {
		t.Error("expected cmd.process.SysProcAttr.Setpgid to be true")
	}
}

func TestCommandContext(t *testing.T) {
	tests := []struct {
		name                    string
		ctx                     context.Context
		expectCommandContextErr bool
	}{
		{
			name: "nominal behavior",
			ctx:  context.Background(),
		},
		{
			name:                    "nil context",
			expectCommandContextErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, err := CommandContext(tc.ctx, zap.NewNop(), "foo")

			if err == nil && !cmd.process.SysProcAttr.Setpgid {
				t.Fatal("expected cmd.process.SysProcAttr.Setpgid to be true")
			}

			if tc.expectCommandContextErr && err == nil {
				t.Error("expected error from CommandContext(), but got none")
			}

			if !tc.expectCommandContextErr && err != nil {
				t.Errorf("expected no error from CommandContext(), but got: %v", err)
			}
		})
	}
}

func TestCmd_Start(t *testing.T) {
	tests := []struct {
		name           string
		cmd            Cmd
		expectStartErr bool
	}{
		{
			name: "nominal behavior",
			cmd:  Command(zap.NewNop(), "echo", "Hello", "World"),
		},
		{
			name:           "start error",
			cmd:            Command(zap.NewNop(), "foo"),
			expectStartErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.Start()

			if tc.expectStartErr && err == nil {
				t.Error("expected error from cmd.Start(), but got none")
			}

			if !tc.expectStartErr && err != nil {
				t.Errorf("expected no error from cmd.Start(), but got: %v", err)
			}
		})
	}
}

func TestCmd_Wait(t *testing.T) {
	tests := []struct {
		name          string
		cmd           Cmd
		expectWaitErr bool
	}{
		{
			name: "nominal behavior",
			cmd: func() Cmd {
				cmd := Command(zap.NewNop(), "echo", "Hello", "World")

				err := cmd.Start()
				if err != nil {
					t.Fatalf("expected no error from cmd.Start(), but got: %v", err)
				}

				return cmd
			}(),
		},
		{
			name: "wait error",
			cmd: func() Cmd {
				cmd := Command(zap.NewNop(), "echo", "Hello", "World")

				err := cmd.Start()
				if err != nil {
					t.Fatalf("expected no error from cmd.Start(), but got: %v", err)
				}

				err = cmd.Kill()
				if err != nil {
					t.Fatalf("expected no error from cmd.Kill(), but got: %v", err)
				}

				return cmd
			}(),
			expectWaitErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.Wait()

			if tc.expectWaitErr && err == nil {
				t.Error("expected error from cmd.Wait(), but got none")
			}

			if !tc.expectWaitErr && err != nil {
				t.Errorf("expected no error from cmd.Wait(), but got: %v", err)
			}
		})
	}
}

func TestCmd_Exec(t *testing.T) {
	tests := []struct {
		name          string
		cmd           Cmd
		timeout       time.Duration
		expectExecErr bool
	}{
		{
			name: "nominal behavior",
			cmd: func() Cmd {
				cmd, err := CommandContext(context.Background(), zap.NewNop(), "echo", "Hello", "World")
				if err != nil {
					t.Fatalf("expected no error from CommandContext(), but got: %v", err)
				}

				return cmd
			}(),
		},
		{
			name:          "nil context",
			cmd:           Command(zap.NewNop(), "echo", "Hello", "World"),
			expectExecErr: true,
		},
		{
			name: "start error",
			cmd: func() Cmd {
				cmd, err := CommandContext(context.Background(), zap.NewNop(), "foo")
				if err != nil {
					t.Fatalf("expected no error from CommandContext(), but got: %v", err)
				}

				return cmd
			}(),
			expectExecErr: true,
		},
		{
			name:          "context done",
			cmd:           Command(zap.NewNop(), "sleep", "2"),
			timeout:       time.Duration(1) * time.Second,
			expectExecErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.timeout > 0 {
				ctx, cancel := context.WithTimeout(context.TODO(), tc.timeout)
				defer cancel()

				tc.cmd.ctx = ctx
			}

			_, err := tc.cmd.Exec()

			if tc.expectExecErr && err == nil {
				t.Error("expected error from cmd.Exec(), but got none")
			}

			if !tc.expectExecErr && err != nil {
				t.Errorf("expected no error from cmd.Exec(), but got: %v", err)
			}
		})
	}
}

func TestCmd_pipeOutput(t *testing.T) {
	tests := []struct {
		name                string
		cmd                 Cmd
		run                 bool
		expectPipeOutputErr bool
	}{
		{
			name: "nominal behavior",
			cmd:  Command(zap.NewExample(), "echo", "Hello", "World"),
			run:  true,
		},
		{
			name: "no debug, no pipe",
			cmd:  Command(zap.NewNop(), "echo", "Hello", "World"),
		},
		{
			name: "stdout already piped",
			cmd: func() Cmd {
				cmd := Command(zap.NewExample(), "echo", "Hello", "World")

				_, err := cmd.process.StdoutPipe()
				if err != nil {
					t.Fatalf("expected no error from cmd.process.StdoutPipe(), but got: %v", err)
				}

				return cmd
			}(),
			expectPipeOutputErr: true,
		},
		{
			name: "stderr already piped",
			cmd: func() Cmd {
				cmd := Command(zap.NewExample(), "echo", "Hello", "World")

				_, err := cmd.process.StderrPipe()
				if err != nil {
					t.Fatalf("expected no error from cmd.process.StderrPipe(), but got: %v", err)
				}

				return cmd
			}(),
			expectPipeOutputErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.pipeOutput()

			if tc.run {
				errStart := tc.cmd.process.Start()
				if errStart != nil {
					t.Fatalf("expected no error from tc.cmd.process.Start(), but got: %v", errStart)
				}
			}

			if tc.expectPipeOutputErr && err == nil {
				t.Error("expected error from cmd.pipeOutput(), but got none")
			}

			if !tc.expectPipeOutputErr && err != nil {
				t.Errorf("expected no error from cmd.pipeOutput(), but got: %v", err)
			}
		})
	}
}

func TestCmd_Kill(t *testing.T) {
	tests := []struct {
		name string
		cmd  Cmd
	}{
		{
			name: "nominal behavior",
			cmd: func() Cmd {
				cmd := Command(zap.NewNop(), "sleep", "60")

				err := cmd.process.Start()
				if err != nil {
					t.Fatalf("expected no error from cmd.process.Start(), but got: %v", err)
				}

				return cmd
			}(),
		},
		{
			name: "no process",
			cmd:  Cmd{logger: zap.NewNop()},
		},
		{
			name: "process already killed",
			cmd: func() Cmd {
				cmd := Command(zap.NewNop(), "sleep", "60")

				err := cmd.process.Start()
				if err != nil {
					t.Fatalf("expected no error from cmd.process.Start(), but got: %v", err)
				}

				err = cmd.Kill()
				if err != nil {
					t.Fatalf("expected no error from cmd.Kill(), but got: %v", err)
				}

				return cmd
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cmd.Kill()
			if err != nil {
				t.Errorf("expected no error from cmd.Kill(), but got: %v", err)
			}
		})
	}
}
