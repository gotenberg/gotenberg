package gotenberg

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGarbageCollect(t *testing.T) {
	t.Run("root path does not exist", func(t *testing.T) {
		err := GarbageCollect(context.Background(), slog.New(slog.DiscardHandler), filepath.Join(t.TempDir(), "missing"), []string{"*"}, time.Now())
		if err == nil {
			t.Fatal("expected error but got none")
		}
	})

	t.Run("remove the expired entries matching a pattern", func(t *testing.T) {
		root := t.TempDir()
		for _, name := range []string{"lu881b8swr.tmp", "OSL_PIPE_1001_SingleOfficeIPC_b2", "notes.tmp.txt", "upload.tmp", "a-request-dir"} {
			err := os.Mkdir(filepath.Join(root, name), 0o755)
			if err != nil {
				t.Fatalf("create %s: %v", name, err)
			}
		}

		// Not expired: created after the expiration time.
		expiration := time.Now()
		fresh := filepath.Join(root, "lu999fresh.tmp")
		err := os.Mkdir(fresh, 0o755)
		if err != nil {
			t.Fatalf("create fresh entry: %v", err)
		}
		future := expiration.Add(time.Minute)
		err = os.Chtimes(fresh, future, future)
		if err != nil {
			t.Fatalf("touch fresh entry: %v", err)
		}

		past := expiration.Add(-time.Minute)
		for _, name := range []string{"lu881b8swr.tmp", "OSL_PIPE_1001_SingleOfficeIPC_b2", "notes.tmp.txt", "upload.tmp", "a-request-dir"} {
			err = os.Chtimes(filepath.Join(root, name), past, past)
			if err != nil {
				t.Fatalf("touch %s: %v", name, err)
			}
		}

		err = GarbageCollect(context.Background(), slog.New(slog.DiscardHandler), root, []string{"lu*.tmp", "OSL_PIPE_*"}, expiration)
		if err != nil {
			t.Fatalf("expected no error but got: %v", err)
		}

		for name, expectExists := range map[string]bool{
			"lu881b8swr.tmp":                   false,
			"OSL_PIPE_1001_SingleOfficeIPC_b2": false,
			"lu999fresh.tmp":                   true,
			// A substring match would have taken these.
			"notes.tmp.txt": true,
			"upload.tmp":    true,
			"a-request-dir": true,
		} {
			_, err = os.Stat(filepath.Join(root, name))
			if exists := err == nil; exists != expectExists {
				t.Errorf("%s exists = %t, want %t", name, exists, expectExists)
			}
		}
	})
}
