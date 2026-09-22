package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Jinnrry/pmail/config"
	"github.com/Jinnrry/pmail/db"
	"github.com/Jinnrry/pmail/signal"
)

const (
	integrationDeadline = 4 * time.Minute
	integrationMarker   = ".pmail-integration-root"
)

// A supervisor process owns the temporary directory. Even os.Exit, a panic in a
// server goroutine, or testing's timeout cannot bypass its fixture cleanup.
func runIntegrationSupervisor() int {
	for _, arg := range os.Args[1:] {
		if arg == "mysql" || arg == "postgres" {
			fmt.Fprintln(os.Stderr, "legacy integration fixture only supports its private SQLite database; external/shared database arguments are rejected")
			return 1
		}
	}
	source, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	root, err := os.MkdirTemp("", "pmail-integration-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(root)
	if err = prepareIntegrationRoot(source, root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	binary, err := filepath.Abs(os.Args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationDeadline)
	defer cancel()
	child := exec.CommandContext(ctx, binary, os.Args[1:]...)
	child.Dir = root
	child.Env = integrationEnvironment(os.Environ(), root)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.WaitDelay = 5 * time.Second
	if err = child.Run(); err != nil {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "integration deadline exceeded; child stopped and temporary files removed")
			return 1
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func integrationEnvironment(environ []string, root string) []string {
	result := make([]string, 0, len(environ)+3)
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "PMail_ROOT", "PMAIL_INTEGRATION_WORKER", "setup_port":
			continue
		}
		result = append(result, entry)
	}
	return append(result, "PMail_ROOT="+root, "PMAIL_INTEGRATION_WORKER=1", "setup_port="+strconv.Itoa(TestPort))
}

func prepareIntegrationRoot(source, root string) error {
	// Only known repository development fixtures are copied, never config.json,
	// database files, plugin configuration or arbitrary directory contents.
	for _, relative := range []string{"config/dkim/dkim.priv", "config/dkim/dkim.public", "config/ssl/private.key", "config/ssl/public.crt"} {
		data, err := os.ReadFile(filepath.Join(source, relative))
		if err != nil {
			return fmt.Errorf("read development fixture %s: %w", relative, err)
		}
		target := filepath.Join(root, relative)
		if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(target, data, 0600); err != nil {
			return err
		}
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, integrationMarker), []byte(canonicalRoot), 0600)
}

func validateIntegrationRoot(root, configured string) error {
	configuredRoot, pathErr := filepath.EvalSymlinks(configured)
	actualRoot, actualErr := filepath.EvalSymlinks(root)
	if pathErr != nil || actualErr != nil || configuredRoot != actualRoot || !strings.HasPrefix(filepath.Base(actualRoot), "pmail-integration-") {
		return errors.New("integration root isolation mismatch")
	}
	marker, err := os.ReadFile(filepath.Join(actualRoot, integrationMarker))
	if err != nil || string(marker) != actualRoot {
		return errors.New("integration worker requires a supervisor-created temporary fixture")
	}
	return nil
}

func runIntegrationWorker(m *testing.M) int {
	root, err := os.Getwd()
	if err == nil {
		err = validateIntegrationRoot(root, config.ROOT_PATH)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	// Fail before starting any listeners if a real service already owns a port.
	for _, port := range []int{TestPort, 25, 465, 587, 110, 995, 993} {
		listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err != nil {
			fmt.Fprintf(os.Stderr, "integration port %d unavailable: %v\n", port, err)
			return 1
		}
		listener.Close()
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	httpClient = &http.Client{Jar: jar, Timeout: 15 * time.Second}
	http.DefaultClient = &http.Client{Timeout: 15 * time.Second}
	finished := make(chan struct{})
	go func() { defer close(finished); main() }()
	ready := time.NewTimer(20 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ready.Stop()
	defer ticker.Stop()
readyLoop:
	for {
		select {
		case <-finished:
			fmt.Fprintln(os.Stderr, "integration server exited before readiness")
			return 1
		case <-ready.C:
			fmt.Fprintln(os.Stderr, "integration setup server readiness timeout")
			return 1
		case <-ticker.C:
			probe, err := http.Get(TestHost + "/api/ping")
			if err == nil {
				io.Copy(io.Discard, probe.Body)
				probe.Body.Close()
				if probe.StatusCode < 500 {
					break readyLoop
				}
			}
		}
	}
	code := m.Run()
	// StopChan may have no receiver if setup failed. Shutdown is always bounded;
	// exiting this worker closes its remaining listeners before parent cleanup.
	select {
	case signal.StopChan <- true:
	case <-finished:
	case <-time.After(2 * time.Second):
	}
	select {
	case <-finished:
		if db.Instance != nil {
			db.Instance.Close()
		}
	case <-time.After(3 * time.Second):
	}
	return code
}

func TestIntegrationFixtureIsolation(t *testing.T) {
	if os.Getenv("PMAIL_INTEGRATION_WORKER") == "1" {
		t.Skip("supervisor-only pure fixture test")
	}
	source := t.TempDir()
	root, err := os.MkdirTemp(t.TempDir(), "pmail-integration-")
	if err != nil {
		t.Fatal(err)
	}
	fixtures := []string{"config/dkim/dkim.priv", "config/dkim/dkim.public", "config/ssl/private.key", "config/ssl/public.crt"}
	for _, relative := range append(fixtures, "config/config.json", "config/pmail.db") {
		path := filepath.Join(source, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(relative), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := prepareIntegrationRoot(source, root); err != nil {
		t.Fatal(err)
	}
	if err := validateIntegrationRoot(root, root); err != nil {
		t.Fatal(err)
	}
	if err := validateIntegrationRoot(source, source); err == nil {
		t.Fatal("accepted non-fixture working directory")
	}
	if err := validateIntegrationRoot(root, source); err == nil {
		t.Fatal("accepted inherited external configuration root")
	}
	if err := os.Remove(filepath.Join(root, integrationMarker)); err != nil {
		t.Fatal(err)
	}
	if err := validateIntegrationRoot(root, root); err == nil {
		t.Fatal("accepted worker without supervisor marker")
	}
	for _, relative := range []string{"config/config.json", "config/pmail.db"} {
		if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("copied user state %s", relative)
		}
		if data, err := os.ReadFile(filepath.Join(source, relative)); err != nil || string(data) != relative {
			t.Fatalf("modified original %s", relative)
		}
	}
	env := integrationEnvironment([]string{"PMail_ROOT=/production", "setup_port=25", "PMAIL_INTEGRATION_WORKER=0", "KEEP=1"}, root)
	for _, entry := range env {
		if strings.Contains(entry, "/production") || entry == "setup_port=25" {
			t.Fatalf("unsafe environment: %s", entry)
		}
	}
}
