package main

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestMainProcess(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "1" {
		main()
		return
	}

	// Case 1: Normal startup and graceful shutdown using SIGINT
	cmd := exec.Command(os.Args[0], "-test.run=TestMainProcess")
	cmd.Env = append(os.Environ(),
		"BE_CRASHER=1",
		"SEERR_API_KEY=test-key-1234567890",
		"LIMBO_PORT=0", // automatic port selection
		"SQLITE_PATH=test_main_ok.db",
		"SCAN_INTERVAL_MINUTES=60", // long interval to avoid periodic scan triggers in test
	)
	err := cmd.Start()
	if err != nil {
		t.Fatalf("failed to start cmd: %v", err)
	}

	// Give the process a short duration to startup and bind port
	time.Sleep(500 * time.Millisecond)

	// Send SIGINT to trigger graceful shutdown
	err = cmd.Process.Signal(syscall.SIGINT)
	if err != nil {
		t.Fatalf("failed to send SIGINT: %v", err)
	}

	// Wait for the process to exit
	err = cmd.Wait()
	if err != nil {
		t.Fatalf("process exited with error: %v", err)
	}

	// Clean up DB
	os.Remove("test_main_ok.db")
}

func TestMainProcessConfigError(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "2" {
		main()
		return
	}

	// Case 2: Config validation failure (missing SEERR_API_KEY)
	cmd := exec.Command(os.Args[0], "-test.run=TestMainProcessConfigError")
	cmd.Env = append(os.Environ(),
		"BE_CRASHER=2",
		"SEERR_API_KEY=", // empty key
	)
	err := cmd.Start()
	if err != nil {
		t.Fatalf("failed to start cmd: %v", err)
	}

	// Expect the process to exit with an error status (1)
	err = cmd.Wait()
	if err == nil {
		t.Fatalf("expected process to fail due to missing SEERR_API_KEY but it exited successfully")
	}
}

func TestMainProcessDBError(t *testing.T) {
	if os.Getenv("BE_CRASHER") == "3" {
		main()
		return
	}

	// Case 3: DB initialization failure (unsupported driver)
	cmd := exec.Command(os.Args[0], "-test.run=TestMainProcessDBError")
	cmd.Env = append(os.Environ(),
		"BE_CRASHER=3",
		"SEERR_API_KEY=test-key-1234567890",
		"DB_DRIVER=invalid_driver_name",
	)
	err := cmd.Start()
	if err != nil {
		t.Fatalf("failed to start cmd: %v", err)
	}

	// Expect the process to exit with an error status (1)
	err = cmd.Wait()
	if err == nil {
		t.Fatalf("expected process to fail due to invalid DB driver but it exited successfully")
	}
}
