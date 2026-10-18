package e2e

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestSecureMockServerWithCLI(t *testing.T) {
	root := repositoryRoot(t)
	port := freePort(t)
	tempDir := t.TempDir()
	serverBinary := filepath.Join(tempDir, "secure-mockserver")
	cliBinary := filepath.Join(tempDir, "threatsim")
	policyPath := filepath.Join(tempDir, "rate-limit.yaml")

	build(t, root, serverBinary, "./examples/secure_mockserver")
	build(t, root, cliBinary, ".")

	policy := `version: "1.0"
simulations:
  - name: "Secure mock server rate limit"
    plugin: "rate_limit"
    config:
      path: "/api/search"
      method: "GET"
      num_requests: 20
      concurrency: 10
      expected_status_code: 429
      expected_body_contains: "rate limit"
`
	if err := os.WriteFile(policyPath, []byte(policy), 0600); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	server := exec.Command(serverBinary)
	server.Dir = root
	server.Env = append(os.Environ(), fmt.Sprintf("THREATSIM_MOCKSERVER_PORT=%d", port))
	if err := server.Start(); err != nil {
		t.Fatalf("start secure mock server: %v", err)
	}
	defer func() {
		_ = server.Process.Kill()
		_ = server.Wait()
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForServer(t, baseURL+"/api/search")

	run := exec.Command(cliBinary, "run", "-t", baseURL, "-f", policyPath)
	run.Dir = root
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("CLI failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "Passed Controls:") || !strings.Contains(string(output), "100.00%") {
		t.Fatalf("expected one passed control, got:\n%s", output)
	}
}

func build(t *testing.T, root, output, packagePath string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", output, packagePath)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", packagePath, err, output)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to locate test file")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(file), "../.."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate test port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func waitForServer(t *testing.T, endpoint string) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(endpoint)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server did not become ready: %s", endpoint)
}
