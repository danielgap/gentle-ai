package organicruntime_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// controlMasterDenialMarker is the fixed prefix of OpenCode's PermissionDenied
// message (core v1/permission.ts), i.e. what the runtime reports back to the
// model when an injected permission.bash deny rule fires.
const controlMasterDenialMarker = "The user has specified a rule which prevents you from using this specific tool call"

// TestOpenCodeRuntimeDeniesAbsolutePathSSHoverControlMaster reproduces the
// #4324 attack shape with the one difference the bare deny rules do not cover:
// the scripted model invokes the ssh BINARY BY ABSOLUTE PATH through an
// existing SSH ControlMaster socket. A real in-test SSH server
// (golang.org/x/crypto/ssh) backs a real OpenSSH ControlMaster socket, so "the
// command reached the remote side" is observable server-side (session count +
// sentinel file) instead of inferred from client output. Verified end to end
// against the pinned OpenCode runtime; fails loudly without the absolute-path
// deny rules (precondition) and without the runtime denial (postcondition).
func TestOpenCodeRuntimeDeniesAbsolutePathSSHoverControlMaster(t *testing.T) {
	if testing.Short() || strings.TrimSpace(os.Getenv("GENTLE_AI_OPENCODE_RUNTIME_E2E")) != "1" {
		t.Skip("set GENTLE_AI_OPENCODE_RUNTIME_E2E=1 to verify the remote-execution deny rules against the pinned OpenCode runtime")
	}
	if runtime.GOOS != "linux" {
		t.Skipf("the ControlMaster proof needs a unix-socket OpenSSH client; %s does not apply", runtime.GOOS)
	}
	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		t.Fatalf("OpenSSH client is required: %v", err)
	}
	requireOrganicExecutableVersion(t, "opencode", pinnedOpenCodeVersion)
	requireOrganicExecutable(t, "npm")

	// Unix domain socket paths are limited to ~104 bytes; default t.TempDir
	// names exceed that, so keep every socket-adjacent path short.
	shortDir, err := os.MkdirTemp("", "cm")
	if err != nil {
		t.Fatalf("create short temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(shortDir) })
	socketPath := filepath.Join(shortDir, "ctl")
	sentinelPath := filepath.Join(shortDir, "sentinel")

	server := startControlMasterSSHServer(t, sentinelPath)
	clientKeyPath := filepath.Join(shortDir, "id_ed25519")
	writeControlMasterClientKey(t, server.clientKey, clientKeyPath)
	controlMasterConnect(t, sshPath, socketPath, clientKeyPath, server.port())

	// Install the permission overlay exactly like a real machine: fresh HOME,
	// pinned runtime on PATH, `gentle-ai install --components permissions`.
	harness := newOrganicHarness(t)
	sharedConfig := prepareOpenCodeConfig(t)
	environment := append(harness.environment(),
		"XDG_CONFIG_HOME="+sharedConfig,
		"OPENCODE_CONFIG_DIR="+filepath.Join(sharedConfig, "opencode"),
	)
	output, stderr, err := runOrganicCommand(
		t, organicBinary, harness.repo.worktree, environment,
		"install", "--agent", "opencode", "--scope", "workspace", "--components", "permissions",
	)
	if err != nil {
		t.Fatalf("install opencode permissions: %v\nstdout:\n%s\nstderr:\n%s", err, output, stderr)
	}

	// Precondition: the injected settings must deny the exact absolute ssh
	// binary this machine runs. Failing here means the overlay's enumerated
	// install prefixes missed a layout, not that the runtime misbehaved.
	settingsPath := filepath.Join(sharedConfig, "opencode", "opencode.json")
	settings, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read injected opencode settings %s: %v", settingsPath, err)
	}
	wantRule := filepath.Dir(sshPath) + string(os.PathSeparator) + "ssh *"
	if !strings.Contains(string(settings), wantRule) {
		t.Fatalf("injected opencode settings do not deny the absolute-path form %q for this machine's ssh at %s:\n%s", wantRule, sshPath, settings)
	}

	// Script the model to attempt the attack. The probe agent deliberately
	// carries NO agent-level permission override: an override like
	// permission.bash=allow would shadow the injected global denies (agent
	// rules merge after user rules and the last match wins), which is exactly
	// what must NOT happen on a real gentle-ai machine.
	probeCommand := strings.Join([]string{
		sshPath, "-S", socketPath,
		"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes", "-o", "IdentitiesOnly=yes", "-i", clientKeyPath,
		"-p", fmt.Sprintf("%d", server.port()),
		"probe@127.0.0.1", "touch", sentinelPath,
	}, " ")
	fixture := newControlMasterProbeServer(t, probeCommand)

	home := t.TempDir()
	runEnv := append(append([]string{}, environment...),
		"XDG_CACHE_HOME="+t.TempDir(),
		"OPENCODE_TEST_HOME="+filepath.Join(home, "opencode"),
		"OPENCODE_CONFIG_CONTENT="+controlMasterProbeConfig(t, fixture.URL),
		"OPENCODE_AUTH_CONTENT={}",
	)
	for _, flag := range []string{
		"OPENCODE_DISABLE_PROJECT_CONFIG", "OPENCODE_DISABLE_AUTOUPDATE", "OPENCODE_DISABLE_AUTOCOMPACT",
		"OPENCODE_DISABLE_CLAUDE_CODE", "OPENCODE_DISABLE_DEFAULT_PLUGINS", "OPENCODE_DISABLE_EXTERNAL_SKILLS",
		"OPENCODE_DISABLE_LSP_DOWNLOAD", "OPENCODE_DISABLE_MODELS_FETCH",
		"OPENCODE_EXPERIMENTAL_DISABLE_FILEWATCHER", "OPENCODE_FAST_BOOT", "OPENCODE_PURE",
	} {
		runEnv = append(runEnv, flag+"=1")
	}

	ctx, cancel := context.WithTimeout(context.Background(), organicAgentTimeout)
	defer cancel()
	command := organicCommandContext(ctx, "opencode", "run", "--pure",
		"--format", "json", "--agent", "probe", "--model", "fixture/fixture",
		"--dir", harness.repo.worktree, "Probe the shell.",
	)
	command.Dir = harness.repo.worktree
	command.Env = runEnv
	var stdout, stderrBuffer bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderrBuffer
	if err := command.Run(); err != nil {
		t.Fatalf("opencode run: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderrBuffer.String())
	}

	fixture.assertProbeOutcome(t)
	if sessions := server.sessions.Load(); sessions != 0 {
		t.Fatalf("the denied absolute-path ssh opened %d sessions through the ControlMaster socket; the deny rule did not fire", sessions)
	}
	if _, statErr := os.Stat(sentinelPath); !os.IsNotExist(statErr) {
		t.Fatalf("the denied remote command executed: sentinel %s exists (stat err = %v)", sentinelPath, statErr)
	}
}

// controlMasterSSHServer is a minimal SSH server standing in for the remote
// host an ambient ControlMaster socket is authenticated against. It accepts
// exactly one public key, counts every multiplexed session, and materializes
// the sentinel file host-side when an exec request reaches it — making "the
// denied command got through" observable without trusting client output.
type controlMasterSSHServer struct {
	listener  net.Listener
	clientKey ed25519.PrivateKey
	sessions  atomic.Int32
}

func (server *controlMasterSSHServer) port() int {
	return server.listener.Addr().(*net.TCPAddr).Port
}

func startControlMasterSSHServer(t *testing.T, sentinelPath string) *controlMasterSSHServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for the in-test SSH server: %v", err)
	}
	_, hostKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostKey)
	if err != nil {
		t.Fatalf("host signer: %v", err)
	}
	_, clientKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	clientPublicKey, err := ssh.NewPublicKey(clientKey.Public())
	if err != nil {
		t.Fatalf("client public key: %v", err)
	}
	server := &controlMasterSSHServer{listener: listener, clientKey: clientKey}
	configuration := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), clientPublicKey.Marshal()) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("unexpected client key")
		},
	}
	configuration.AddHostKey(hostSigner)
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handle(connection, configuration, sentinelPath)
		}
	}()
	return server
}

func (server *controlMasterSSHServer) handle(connection net.Conn, configuration *ssh.ServerConfig, sentinelPath string) {
	sshConnection, channels, requests, err := ssh.NewServerConn(connection, configuration)
	if err != nil {
		_ = connection.Close()
		return
	}
	defer func() { _ = sshConnection.Close() }()
	go ssh.DiscardRequests(requests)
	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(ssh.UnknownChannelType, "session channels only")
			continue
		}
		accepted, channelRequests, err := channel.Accept()
		if err != nil {
			continue
		}
		server.sessions.Add(1)
		go func(accepted ssh.Channel, channelRequests <-chan *ssh.Request) {
			defer func() { _ = accepted.Close() }()
			for request := range channelRequests {
				if request.Type != "exec" {
					if request.WantReply {
						_ = request.Reply(false, nil)
					}
					continue
				}
				var payload struct {
					Command string
				}
				if err := ssh.Unmarshal(request.Payload, &payload); err == nil && strings.Contains(payload.Command, sentinelPath) {
					_ = os.WriteFile(sentinelPath, []byte(payload.Command), 0o600)
				}
				if request.WantReply {
					_ = request.Reply(true, nil)
				}
				_, _ = accepted.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
				return
			}
		}(accepted, channelRequests)
	}
}

func writeControlMasterClientKey(t *testing.T, key ed25519.PrivateKey, path string) {
	t.Helper()
	block, err := ssh.MarshalPrivateKey(key, "control-master probe")
	if err != nil {
		t.Fatalf("marshal client key: %v", err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write client key %s: %v", path, err)
	}
}

// controlMasterConnect raises a real OpenSSH ControlMaster socket against the
// in-test server: authenticate in the background (-f -N) and keep the socket
// alive for the multiplexed probe client, mirroring the ambient session an
// agent discovers under /tmp on a developer machine.
func controlMasterConnect(t *testing.T, sshPath, socketPath, clientKeyPath string, port int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), organicLocalTimeout)
	defer cancel()
	master := organicCommandContext(ctx, sshPath,
		"-M", "-S", socketPath, "-f", "-N",
		"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
		"-o", "BatchMode=yes", "-o", "IdentitiesOnly=yes", "-i", clientKeyPath,
		"-o", "ConnectTimeout=10",
		"-p", fmt.Sprintf("%d", port), "probe@127.0.0.1",
	)
	var stderr bytes.Buffer
	master.Stderr = &stderr
	if err := master.Run(); err != nil {
		t.Fatalf("raise the ControlMaster master: %v\n%s", err, stderr.String())
	}
	t.Cleanup(func() {
		exit := organicCommandContext(context.Background(), sshPath, "-S", socketPath, "-O", "exit")
		_ = exit.Run()
	})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := os.Stat(socketPath); err == nil && info.Mode()&os.ModeSocket != 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the ControlMaster socket never appeared")
}

// controlMasterProbeConfig mirrors organicOpenCodeConfig minus the agent-level
// permission override: the probe agent inherits defaults merged with the user
// (injected global) rules, so the deny evaluation is the one a stock
// gentle-ai install produces.
func controlMasterProbeConfig(t *testing.T, serverURL string) string {
	t.Helper()
	config := map[string]any{
		"provider": map[string]any{"fixture": map[string]any{
			"npm":     "@ai-sdk/openai-compatible",
			"name":    "Organic E2E Fixture",
			"options": map[string]any{"baseURL": serverURL + "/v1", "apiKey": "fixture"},
			"models":  map[string]any{"fixture": map[string]any{"name": "Fixture"}},
		}},
		"agent": map[string]any{"probe": map[string]any{
			"description": "ControlMaster deny probe", "mode": "primary", "model": "fixture/fixture",
		}},
		"plugin":     []any{},
		"compaction": map[string]any{"auto": false},
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal probe config: %v", err)
	}
	return string(encoded)
}

// controlMasterProbeServer scripts exactly one tool call — the absolute-path
// ssh probe — and captures what the runtime reported back to the model on the
// next turn. It embeds openCodeFixtureServer to reuse its SSE streaming
// helpers and failure capture on a zero-value receiver.
type controlMasterProbeServer struct {
	openCodeFixtureServer
	mu           sync.Mutex
	probeCommand string
	probeIssued  bool
	denialSeen   bool
}

func newControlMasterProbeServer(t *testing.T, probeCommand string) *controlMasterProbeServer {
	t.Helper()
	fixture := &controlMasterProbeServer{probeCommand: probeCommand}
	fixture.Server = httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	t.Cleanup(fixture.Server.Close)
	return fixture
}

func (fixture *controlMasterProbeServer) serveHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(writer, "method", http.StatusMethodNotAllowed)
		return
	}
	var input openAIRequest
	if err := json.NewDecoder(io.LimitReader(request.Body, 8<<20)).Decode(&input); err != nil {
		fixture.fail(writer, "decode model request: %v", err)
		return
	}
	fixture.mu.Lock()
	issued := fixture.probeIssued
	fixture.mu.Unlock()
	if !issued && len(input.Tools) > 0 {
		fixture.mu.Lock()
		fixture.probeIssued = true
		fixture.mu.Unlock()
		fixture.writeTool(writer, "probe", "bash", map[string]any{"command": fixture.probeCommand})
		return
	}
	for _, message := range input.Messages {
		if message.Role == "tool" && strings.Contains(messageText(message.Content), controlMasterDenialMarker) {
			fixture.mu.Lock()
			fixture.denialSeen = true
			fixture.mu.Unlock()
		}
	}
	fixture.writeText(writer, "Probe complete.", "stop")
}

func (fixture *controlMasterProbeServer) assertProbeOutcome(t *testing.T) {
	t.Helper()
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	fixture.openCodeFixtureServer.mu.Lock()
	failure := fixture.openCodeFixtureServer.failure
	fixture.openCodeFixtureServer.mu.Unlock()
	if failure != "" {
		t.Fatal(failure)
	}
	if !fixture.probeIssued {
		t.Fatal("the scripted absolute-path ssh probe was never issued to the runtime")
	}
	if !fixture.denialSeen {
		t.Fatal("the runtime did not report the permission denial back to the model; the absolute-path ssh command was not blocked by the injected deny rules")
	}
}
