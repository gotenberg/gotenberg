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
		scenario                  string
		ctx                       context.Context
		expectCommandContextError bool
	}{
		{
			scenario:                  "nominal behavior",
			ctx:                       context.Background(),
			expectCommandContextError: false,
		},
		{
			scenario:                  "nil context",
			ctx:                       nil,
			expectCommandContextError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			cmd, err := CommandContext(tc.ctx, zap.NewNop(), "foo")

			if err == nil && !cmd.process.SysProcAttr.Setpgid {
				t.Fatal("expected cmd.process.SysProcAttr.Setpgid to be true")
			}

			if !tc.expectCommandContextError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectCommandContextError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestCmd_Start(t *testing.T) {
	tests := []struct {
		scenario         string
		cmd              *Cmd
		expectStartError bool
	}{
		{
			scenario:         "nominal behavior",
			cmd:              Command(zap.NewNop(), "echo", "Hello", "World"),
			expectStartError: false,
		},
		{
			scenario:         "start error",
			cmd:              Command(zap.NewNop(), "foo"),
			expectStartError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			err := tc.cmd.Start()

			if !tc.expectStartError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectStartError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestCmd_Wait(t *testing.T) {
	tests := []struct {
		scenario        string
		cmd             *Cmd
		expectWaitError bool
	}{
		{
			scenario: "nominal behavior",
			cmd: func() *Cmd {
				cmd := Command(zap.NewNop(), "echo", "Hello", "World")
				err := cmd.Start()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return cmd
			}(),
			expectWaitError: false,
		},
		{
			scenario: "wait error",
			cmd: func() *Cmd {
				cmd := Command(zap.NewNop(), "echo", "Hello", "World")
				err := cmd.Start()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				err = cmd.Kill()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return cmd
			}(),
			expectWaitError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			err := tc.cmd.Wait()

			if !tc.expectWaitError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectWaitError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestCmd_Exec(t *testing.T) {
	tests := []struct {
		scenario        string
		cmd             *Cmd
		timeout         time.Duration
		expectExecError bool
	}{
		{
			scenario: "nominal behavior",
			cmd: func() *Cmd {
				cmd, err := CommandContext(context.Background(), zap.NewNop(), "echo", "Hello", "World")
				if err != nil {
					t.Fatalf("expected no error from CommandContext(), but got: %v", err)
				}
				return cmd
			}(),
			expectExecError: false,
		},
		{
			scenario:        "nil context",
			cmd:             Command(zap.NewNop(), "echo", "Hello", "World"),
			expectExecError: true,
		},
		{
			scenario: "start error",
			cmd: func() *Cmd {
				cmd, err := CommandContext(context.Background(), zap.NewNop(), "foo")
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return cmd
			}(),
			expectExecError: true,
		},
		{
			scenario:        "context done",
			cmd:             Command(zap.NewNop(), "sleep", "2"),
			timeout:         time.Duration(1) * time.Second,
			expectExecError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			if tc.timeout > 0 {
				ctx, cancel := context.WithTimeout(context.TODO(), tc.timeout)
				defer cancel()

				tc.cmd.ctx = ctx
			}

			_, err := tc.cmd.Exec()

			if !tc.expectExecError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectExecError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestCmd_pipeOutput(t *testing.T) {
	tests := []struct {
		scenario              string
		cmd                   *Cmd
		run                   bool
		expectPipeOutputError bool
	}{
		{
			scenario:              "nominal behavior",
			cmd:                   Command(zap.NewExample(), "echo", "Hello", "World"),
			run:                   true,
			expectPipeOutputError: false,
		},
		{
			scenario:              "no debug, no pipe",
			cmd:                   Command(zap.NewNop(), "echo", "Hello", "World"),
			run:                   false,
			expectPipeOutputError: false,
		},
		{
			scenario: "stdout already piped",
			cmd: func() *Cmd {
				cmd := Command(zap.NewExample(), "echo", "Hello", "World")
				_, err := cmd.process.StdoutPipe()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return cmd
			}(),
			run:                   false,
			expectPipeOutputError: true,
		},
		{
			scenario: "stderr already piped",
			cmd: func() *Cmd {
				cmd := Command(zap.NewExample(), "echo", "Hello", "World")
				_, err := cmd.process.StderrPipe()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return cmd
			}(),
			run:                   false,
			expectPipeOutputError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			err := tc.cmd.pipeOutput()

			if tc.run {
				errStart := tc.cmd.process.Start()
				if errStart != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
			}

			if !tc.expectPipeOutputError && err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if tc.expectPipeOutputError && err == nil {
				t.Fatal("expected error but got none")
			}
		})
	}
}

func TestCmd_Kill(t *testing.T) {
	tests := []struct {
		scenario string
		cmd      *Cmd
	}{
		{
			scenario: "nominal behavior",
			cmd: func() *Cmd {
				cmd := Command(zap.NewNop(), "sleep", "60")
				err := cmd.process.Start()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return cmd
			}(),
		},
		{
			scenario: "no process",
			cmd:      &Cmd{logger: zap.NewNop()},
		},
		{
			scenario: "process already killed",
			cmd: func() *Cmd {
				cmd := Command(zap.NewNop(), "sleep", "60")
				err := cmd.process.Start()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				err = cmd.Kill()
				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}
				return cmd
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.scenario, func(t *testing.T) {
			err := tc.cmd.Kill()
			if err != nil {
				t.Fatalf("expected no error but got: %v", err)
			}
		})
	}
}
