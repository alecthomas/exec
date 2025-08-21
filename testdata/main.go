package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/exec"
)

func main() {
	// Spawn multiple sleep processes
	fmt.Printf("Test program starting with PID %d, PPID %d\n", os.Getpid(), os.Getppid())

	for i := 0; i < 3; i++ {
		cmd := exec.Command("./testdata/script.sh") // Use longer sleep
		err := cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start sleep %d: %v\n", i, err)
			os.Exit(1)
		}
		fmt.Printf("Started sleep process %d with PID %d\n", i, cmd.Process.Pid)
	}

	fmt.Printf("All sleep processes started, waiting...\n")

	// Keep the main process alive
	time.Sleep(60 * time.Second)
}
