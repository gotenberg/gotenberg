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
		t.Error("expected Setpgid to be true")
	}
}

func TestCommandContext(t *testing.T) {
	for i, tc := range []struct {
		ctx       context.Context
		expectErr bool
	}{
		{
			ctx:       nil,
			expectErr: true,
		},
		{
			ctx: context.TODO(),
		},
	} {
		cmd, err := CommandContext(tc.ctx, zap.NewNop(), "foo")

		if err == nil && !cmd.process.SysProcAttr.Setpgid {
			t.Fatalf("test %d: expected Setpgid to be true", i)
		}

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestCmd_Start(t *testing.T) {
	for i, tc := range []struct {
		cmd       Cmd
		expectErr bool
	}{
		{
			cmd:       Command(zap.NewNop(), "foo"),
			expectErr: true,
		},
		{
			cmd: Command(zap.NewNop(), "echo", "Hello", "World"),
		},
	} {
		err := tc.cmd.Start()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestCmd_Exec(t *testing.T) {
	for i, tc := range []struct {
		cmd       Cmd
		timeout   time.Duration
		expectErr bool
	}{
		{
			cmd:       Command(zap.NewNop(), "foo"),
			expectErr: true,
		},
		{
			cmd:       Command(zap.NewNop(), "foo"),
			timeout:   time.Duration(5) * time.Second,
			expectErr: true,
		},
		{
			cmd:     Command(zap.NewNop(), "echo", "Hello", "World"),
			timeout: time.Duration(5) * time.Second,
		},
		{
			cmd:       Command(zap.NewNop(), "sleep", "3"),
			timeout:   time.Duration(2) * time.Second,
			expectErr: true,
		},
	} {
		if tc.timeout > 0 {
			ctx, cancel := context.WithTimeout(context.TODO(), tc.timeout)
			defer cancel()

			tc.cmd.ctx = ctx
		}

		err := tc.cmd.Exec()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestCmd_pipeOutput(t *testing.T) {
	for i, tc := range []struct {
		cmd       Cmd
		run       bool
		expectErr bool
	}{
		{
			cmd: Command(zap.NewNop(), "echo", "Hello", "World"),
		},
		{
			cmd: func() Cmd {
				cmd := Command(zap.NewExample(), "echo", "Hello", "World")
				_, err := cmd.process.StdoutPipe()

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return cmd
			}(),
			expectErr: true,
		},
		{
			cmd: func() Cmd {
				cmd := Command(zap.NewExample(), "echo", "Hello", "World")
				_, err := cmd.process.StderrPipe()

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return cmd
			}(),
			expectErr: true,
		},
		{
			cmd: Command(zap.NewExample(), "echo", "Hello", "World"),
			run: true,
		},
	} {
		err := tc.cmd.pipeOutput()

		if tc.run {
			errStart := tc.cmd.process.Start()

			if errStart != nil {
				t.Fatalf("test %d: expected no error but got: %v", i, err)
			}
		}

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}

func TestCmd_Kill(t *testing.T) {
	for i, tc := range []struct {
		cmd       Cmd
		expectErr bool
	}{
		{
			cmd: Cmd{logger: zap.NewNop()},
		},
		{
			cmd: func() Cmd {
				cmd := Command(zap.NewNop(), "sleep", "60")
				err := cmd.process.Start()

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return cmd
			}(),
		},
		{
			cmd: func() Cmd {
				cmd := Command(zap.NewNop(), "echo", "Hello", "World")
				err := cmd.process.Run()

				if err != nil {
					t.Fatalf("expected no error but got: %v", err)
				}

				return cmd
			}(),
		},
	} {
		err := tc.cmd.Kill()

		if tc.expectErr && err == nil {
			t.Errorf("test %d: expected error but got: %v", i, err)
		}

		if !tc.expectErr && err != nil {
			t.Errorf("test %d: expected no error but got: %v", i, err)
		}
	}
}
