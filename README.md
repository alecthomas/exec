# exec

A drop-in replacement for Go's `os/exec` package that guarantees subprocesses terminate when their parent dies.

## Problem

With `os/exec`, if your Go program exits unexpectedly, child processes may continue running as orphans.

## Solution

This package embeds a small C intermediary binary that:
1. Launches your subprocess
2. Monitors the parent Go process
3. Kills the subprocess if the parent dies

## Usage

```go
import "github.com/alecthomas/exec"

// Drop-in replacement for os/exec
cmd := exec.Command("long-running-process")
err := cmd.Run()
```

## Platforms

Supports Linux and macOS on amd64 and arm64.

## Build Requirements

- Zig (for building intermediary binaries)
- Just (for build automation)

Run `just build` to rebuild intermediary binaries.