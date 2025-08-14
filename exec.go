//go:build (linux || darwin) && (amd64 || arm64)

// Package exec is identical to os/exec except that it guarantees that subprocesses will terminate when their parent
// does.
//
// It achieves this by embedding a tiny C binary that is launched as an intermediary, watches the parent PID for
// termination, then terminates the child.
package exec

import (
	"compress/gzip"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

var (
	//go:embed intermediary/*.gz
	binaries      embed.FS
	extracted     sync.Once
	extractedPath string
)

type Cmd = exec.Cmd
type Error = exec.Error
type ExitError = exec.ExitError

var targetMap = map[string]string{
	"arm64-linux":  "aarch64-linux",
	"amd64-linux":  "x86_64-linux",
	"arm64-darwin": "aarch64-macos",
	"amd64-darwin": "x86_64-macos",
}

var (
	ErrDot       = exec.ErrDot
	ErrNotFound  = exec.ErrNotFound
	ErrWaitDelay = exec.ErrWaitDelay
)

func CommandContext(ctx context.Context, name string, arg ...string) *Cmd {
	// Extract the intermediary binary to a temporary file on first use
	extracted.Do(func() {
		if err := extractBinary(); err != nil {
			panic(err)
		}
	})
	return exec.CommandContext(ctx, extractedPath, append([]string{name}, arg...)...)
}

func Command(name string, arg ...string) *Cmd {
	return CommandContext(context.Background(), name, arg...)
}

func LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func extractBinary() error {
	w, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer w.Close() //nolint

	target, ok := targetMap[runtime.GOARCH+"-"+runtime.GOOS]
	if !ok {
		return fmt.Errorf("unsupported architecture %s-%s", runtime.GOARCH, runtime.GOOS)
	}

	r, err := binaries.Open("intermediary/intermediary-" + target + ".gz")
	if err != nil {
		return err
	}
	defer r.Close() //nolint
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, gzr)
	if err != nil {
		return err
	}
	err = w.Chmod(0700)
	if err != nil {
		return err
	}
	extractedPath = w.Name()
	return nil
}
