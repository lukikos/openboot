//go:build e2e && vm

package e2e

import (
	"fmt"
	"strings"
	"testing"

	"github.com/openbootdotdev/openboot/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startMockAPIServerForPush starts a Python HTTP server that handles:
//   - GET any path: serves the remote config JSON (for pull tests)
//   - POST /api/configs/from-snapshot: records upload body to /tmp/push-received.json
//   - PUT /api/configs/from-snapshot: same (used when --slug is set)
func startMockAPIServerForPush(t *testing.T, vm *testutil.TartVM, port int) string {
	t.Helper()

	writeCmd := fmt.Sprintf(`printf '%%s' %s > /tmp/mock-config.json`, shellescape(mockAPIConfig))
	_, err := vm.Run(writeCmd)
	require.NoError(t, err, "write mock config JSON")

	pyServer := fmt.Sprintf(`import http.server, pathlib
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        body = pathlib.Path('/tmp/mock-config.json').read_bytes()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(body)))
        self.end_headers()
        self.wfile.write(body)
    def do_POST(self):
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length)
        pathlib.Path('/tmp/push-received.json').write_bytes(body)
        resp = b'{"slug": "test-config"}'
        self.send_response(201)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(resp)))
        self.end_headers()
        self.wfile.write(resp)
    def do_PUT(self):
        length = int(self.headers.get('Content-Length', 0))
        body = self.rfile.read(length)
        pathlib.Path('/tmp/push-received.json').write_bytes(body)
        resp = b'{"slug": "existing-config"}'
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(resp)))
        self.end_headers()
        self.wfile.write(resp)
    def log_message(self, *a): pass
http.server.HTTPServer(('127.0.0.1', %d), H).serve_forever()
`, port)

	_, err = vm.Run(fmt.Sprintf("cat > /tmp/mock-server-push.py << 'PYEOF'\n%sPYEOF", pyServer))
	require.NoError(t, err, "write mock push server script")

	pidOut, err := vm.Run(fmt.Sprintf("nohup python3 /tmp/mock-server-push.py >/tmp/mock-push-api.log 2>&1 & echo $!"))
	require.NoError(t, err, "start mock push API server")
	pid := strings.TrimSpace(pidOut)
	t.Logf("mock push API server started (pid=%s) on port %d", pid, port)

	// Poll until server responds (up to 10s)
	for i := 0; i < 10; i++ {
		_, _ = vm.Run("sleep 1")
		out, curlErr := vm.Run(fmt.Sprintf("curl -s http://localhost:%d/check", port))
		if curlErr == nil && strings.Contains(out, "agnoster") {
			break
		}
		t.Logf("waiting for push mock server (attempt %d): err=%v", i+1, curlErr)
	}

	t.Cleanup(func() {
		if pid != "" {
			_, _ = vm.Run("kill " + pid + " 2>/dev/null || true")
		}
	})

	return fmt.Sprintf("http://localhost:%d", port)
}

// writeFakeAuth writes a fake auth token to ~/.openboot/auth.json in the VM,
// bypassing the login prompt in push tests.
func writeFakeAuth(t *testing.T, vm *testutil.TartVM, username, token string) {
	t.Helper()
	authJSON := fmt.Sprintf(
		`{"token":%q,"username":%q,"expires_at":"2099-01-01T00:00:00Z","created_at":"2024-01-01T00:00:00Z"}`,
		token, username,
	)
	_, err := vm.Run(fmt.Sprintf(
		"mkdir -p ~/.openboot && printf '%%s' %s > ~/.openboot/auth.json && chmod 600 ~/.openboot/auth.json",
		shellescape(authJSON),
	))
	require.NoError(t, err, "write fake auth token")
}

// TestE2E_Pull_DryRunShowsDiff verifies that `openboot pull --dry-run` is a
// functional alias for `openboot sync --dry-run`: it fetches the remote config
// and shows the shell diff without applying any changes.
func TestE2E_Pull_DryRunShowsDiff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping VM test in short mode")
	}

	vm := testutil.NewTartVM(t)
	installOhMyZsh(t, vm)
	bin := vmCopyDevBinary(t, vm)

	apiURL := startMockAPIServer(t, vm, 19888)
	writeSyncSource(t, vm, "testuser/myconfig")

	env := map[string]string{
		"PATH":             brewPath,
		"OPENBOOT_API_URL": apiURL,
	}
	out, err := vm.RunWithEnv(env, bin+" pull --dry-run")
	t.Logf("pull --dry-run output:\n%s", out)
	if err != nil {
		t.Logf("exit: %v", err)
	}

	assert.Contains(t, out, "Shell Changes", "should show Shell Changes section")
	assert.Contains(t, out, "agnoster", "should show target theme")
	assert.Contains(t, out, "→", "should show change arrows")
}

// TestE2E_Pull_YesFlagAppliesChanges verifies that `openboot pull --yes`
// applies shell config changes non-interactively (no TUI prompts required),
// making it suitable for CI/CD and scripted use.
func TestE2E_Pull_YesFlagAppliesChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping VM test in short mode")
	}

	vm := testutil.NewTartVM(t)
	installOhMyZsh(t, vm)
	bin := vmCopyDevBinary(t, vm)

	apiURL := startMockAPIServer(t, vm, 19889)
	writeSyncSource(t, vm, "testuser/myconfig")

	zshrcBefore, err := vm.Run("cat ~/.zshrc")
	require.NoError(t, err)
	assert.Contains(t, zshrcBefore, "robbyrussell", "initial theme should be robbyrussell")

	env := map[string]string{
		"PATH":             brewPath,
		"OPENBOOT_API_URL": apiURL,
	}
	// --install-only skips the "remove extra packages?" prompt from the base VM image.
	out, err := vm.RunWithEnv(env, bin+" pull --yes --install-only")
	t.Logf("pull --yes output:\n%s", out)
	if err != nil {
		t.Logf("exit: %v", err)
	}

	zshrcAfter, err := vm.Run("cat ~/.zshrc")
	require.NoError(t, err)
	t.Logf("zshrc after pull:\n%s", zshrcAfter)

	assert.Contains(t, zshrcAfter, `ZSH_THEME="agnoster"`, "theme should be updated to agnoster")
	assert.Contains(t, zshrcAfter, "docker", "plugins should include docker")
}

// TestE2E_Push_AutoCapture_UploadSnapshot verifies that `openboot push` (no args)
// captures the current system snapshot and uploads it to openboot.dev.
//
// Setup: fake auth token in ~/.openboot/auth.json, mock API server accepting POST.
// Interaction: huh prompts for config name, description, visibility.
// Expected: server receives a JSON body containing "captured_at" (snapshot format).
func TestE2E_Push_AutoCapture_UploadSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping VM test in short mode")
	}

	vm := testutil.NewTartVM(t)
	bin := vmCopyDevBinary(t, vm)

	apiURL := startMockAPIServerForPush(t, vm, 19890)
	writeFakeAuth(t, vm, "testuser", "fake-token-e2e")

	// Write env vars + command to a shell script to avoid Tcl quoting issues in expect.
	pushScript := fmt.Sprintf("#!/bin/sh\nexport PATH=%q\nexport OPENBOOT_API_URL=%q\nexec %s push\n",
		brewPath, apiURL, bin)
	_, err := vm.Run(fmt.Sprintf("printf '%%s' %s > /tmp/run-push.sh && chmod +x /tmp/run-push.sh",
		shellescape(pushScript)))
	require.NoError(t, err, "write push script")

	output, _ := vm.RunInteractive("/tmp/run-push.sh", []testutil.ExpectStep{
		{Expect: "Config name", Send: "Test Config\r"},
		{Expect: "Description", Send: "\r"},
		{Expect: "Who can see", Send: "\r"},
	}, 120)
	t.Logf("push interactive output:\n%s", output)

	// Verify the mock server received the snapshot upload
	received, err := vm.Run("cat /tmp/push-received.json 2>/dev/null")
	require.NoError(t, err, "server should have saved received body")
	t.Logf("received upload body (first 300 chars): %.300s", received)

	assert.Contains(t, received, "captured_at", "upload should contain snapshot with captured_at")
	assert.Contains(t, received, "snapshot", "request body should wrap the snapshot")
	assert.Contains(t, output, "uploaded successfully", "output should confirm upload success")
}

// TestE2E_Push_AutoCapture_UsesSyncSourceSlug verifies that `openboot push` (no args)
// automatically uses the slug from the saved sync source, sending a PUT request
// to update the existing config rather than creating a new one.
func TestE2E_Push_AutoCapture_UsesSyncSourceSlug(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping VM test in short mode")
	}

	vm := testutil.NewTartVM(t)
	bin := vmCopyDevBinary(t, vm)

	apiURL := startMockAPIServerForPush(t, vm, 19891)
	writeFakeAuth(t, vm, "testuser", "fake-token-e2e")
	// Write sync source with a slug — push should reuse it for in-place update
	_, err := vm.Run(`mkdir -p ~/.openboot && echo '{"user_slug":"testuser/myconfig","username":"testuser","slug":"myconfig"}' > ~/.openboot/sync_source.json`)
	require.NoError(t, err, "write sync source with slug")

	pushScript := fmt.Sprintf("#!/bin/sh\nexport PATH=%q\nexport OPENBOOT_API_URL=%q\nexec %s push\n",
		brewPath, apiURL, bin)
	_, err = vm.Run(fmt.Sprintf("printf '%%s' %s > /tmp/run-push-slug.sh && chmod +x /tmp/run-push-slug.sh",
		shellescape(pushScript)))
	require.NoError(t, err, "write push script")

	output, _ := vm.RunInteractive("/tmp/run-push-slug.sh", []testutil.ExpectStep{
		{Expect: "Config name", Send: "Updated Config\r"},
		{Expect: "Description", Send: "\r"},
		{Expect: "Who can see", Send: "\r"},
	}, 120)
	t.Logf("push with slug output:\n%s", output)

	// The mock server saves PUT body to /tmp/push-received.json same as POST
	received, err := vm.Run("cat /tmp/push-received.json 2>/dev/null")
	require.NoError(t, err, "server should have saved received body")
	t.Logf("received upload body (first 300 chars): %.300s", received)

	// The body should contain config_slug indicating it's an update
	assert.Contains(t, received, "myconfig", "body should reference the existing slug")
	assert.Contains(t, received, "captured_at", "upload should contain snapshot data")
	assert.Contains(t, output, "uploaded successfully", "output should confirm upload success")
}
