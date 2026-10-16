//go:build windows

package applications

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestProcessFixture(t *testing.T) {
	if os.Getenv("FOCUS_LOCK_TEST_HELPER") == "1" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
}
func TestOnlyOwnedDisposableFixtureIsTerminated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestProcessFixture$")
	cmd.Env = append(os.Environ(), "FOCUS_LOCK_TEST_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	p, err := Inspect(uint32(cmd.Process.Pid), uint32(os.Getpid()))
	if err != nil {
		t.Fatal(err)
	}
	if p.Safe {
		t.Skip("Windows protects the test process in this environment (e.g. elevated or session 0)")
	}
	if Allowed(p, nil, nil, true, nil) {
		t.Fatal("unlisted process allowed")
	}
	stale := p
	stale.Created++
	if Terminate(stale) == nil {
		t.Fatal("recycled PID not rejected")
	}
	if err = Terminate(p); err != nil {
		t.Fatal(err)
	}
}
