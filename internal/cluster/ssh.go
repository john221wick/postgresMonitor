package cluster

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	User    string
	Host    string
	Port    int
	KeyPath string
}

// ParseSSHCommand parses raw SSH commands from cloud providers.
// Examples:
//
//	"ssh -p 20544 root@203.0.113.10"
//	"ssh -p 41922 root@ssh5.vast.ai -L 8080:localhost:8080"
//	"ssh root@192.168.1.100"
func ParseSSHCommand(raw string) (*SSHConfig, error) {
	tokens := strings.Fields(strings.TrimSpace(raw))
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty ssh command")
	}

	// Skip leading "ssh" if present
	start := 0
	if tokens[0] == "ssh" {
		start = 1
	}

	config := &SSHConfig{Port: 22}
	var userHost string

	for i := start; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case tok == "-p" && i+1 < len(tokens):
			i++
			p, err := strconv.Atoi(tokens[i])
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", tokens[i])
			}
			config.Port = p
		case tok == "-i" && i+1 < len(tokens):
			i++
			config.KeyPath = tokens[i]
		case tok == "-L" || tok == "-R" || tok == "-D" || tok == "-o" || tok == "-J":
			// Skip flag + its argument
			i++
		case strings.HasPrefix(tok, "-"):
			// Skip unknown flags
		default:
			// This should be user@host
			if userHost == "" {
				userHost = tok
			}
		}
	}

	if userHost == "" {
		return nil, fmt.Errorf("no user@host found in: %s", raw)
	}

	parts := strings.SplitN(userHost, "@", 2)
	if len(parts) == 2 {
		config.User = parts[0]
		config.Host = parts[1]
	} else {
		// Bare hostname or SSH config alias (e.g. "ssh host")
		// Try resolving from ~/.ssh/config, fall back to alias as-is
		resolved := resolveSSHAlias(userHost)
		config.Host = resolved.Host
		if resolved.User != "" {
			config.User = resolved.User
		}
		if resolved.Port != 0 {
			config.Port = resolved.Port
		}
		if resolved.KeyPath != "" && config.KeyPath == "" {
			config.KeyPath = resolved.KeyPath
		}
	}

	// Handle host:port format
	if host, port, err := net.SplitHostPort(config.Host); err == nil {
		config.Host = host
		if p, err := strconv.Atoi(port); err == nil {
			config.Port = p
		}
	}

	return config, nil
}

type SSHSession struct {
	Client      *ssh.Client
	Config      SSHConfig
	LocalPort   int
	forwardDone chan struct{}
	mu          sync.Mutex
}

// Connect establishes an SSH connection.
func Connect(config SSHConfig) (*SSHSession, error) {
	key, err := loadPrivateKey(config.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("load key: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: config.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	return &SSHSession{
		Client:      client,
		Config:      config,
		forwardDone: make(chan struct{}),
	}, nil
}

// RunCommand executes a command on the remote and returns output.
func (s *SSHSession) RunCommand(cmd string) (string, error) {
	session, err := s.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	return string(out), err
}

// RemoteInfo holds auto-detected info about remote machine.
type RemoteInfo struct {
	Hostname string
	Arch     string
	GoArch   string
	OS       string
}

// DetectRemote auto-detects remote machine capabilities.
func (s *SSHSession) DetectRemote() (*RemoteInfo, error) {
	info := &RemoteInfo{}

	// Hostname
	hostnameOut, _ := s.RunCommand("hostname")
	info.Hostname = strings.TrimSpace(hostnameOut)

	// Arch
	archOut, err := s.RunCommand("uname -m")
	if err != nil {
		return nil, fmt.Errorf("detect arch: %w", err)
	}
	info.Arch = strings.TrimSpace(archOut)
	switch info.Arch {
	case "x86_64":
		info.GoArch = "amd64"
	case "aarch64", "arm64":
		info.GoArch = "arm64"
	default:
		return nil, fmt.Errorf("unsupported arch: %s", info.Arch)
	}

	// OS
	osOut, _ := s.RunCommand("cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'\"' -f2")
	info.OS = strings.TrimSpace(osOut)
	if info.OS == "" {
		info.OS = "Linux"
	}

	fmt.Printf("Remote: arch=%s os=%s hostname=%s\n", info.Arch, info.OS, info.Hostname)

	return info, nil
}

// CrossCompileAndSCP builds the monitor agent for the remote arch and copies to remotePath.
func (s *SSHSession) CrossCompileAndSCP(remotePath string) error {
	info, err := s.DetectRemote()
	if err != nil {
		return err
	}

	// Kill any existing agent — we're deploying fresh on each connect
	s.RunCommand("pkill -f 'pgmonitor --agent' 2>/dev/null || true")
	time.Sleep(500 * time.Millisecond)

	// Find project root
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}
	projectRoot := findProjectRoot(execPath)
	if projectRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			projectRoot = findProjectRoot(cwd)
		}
	}
	if projectRoot == "" {
		return fmt.Errorf("could not find project root (go.mod)")
	}

	tmpBin := filepath.Join(os.TempDir(), fmt.Sprintf("pgmonitor-linux-%s", info.GoArch))
	defer os.Remove(tmpBin)

	args := []string{"build", "-o", tmpBin, "./cmd/agent/"}

	buildCmd := exec.Command("go", args...)
	buildCmd.Dir = projectRoot
	buildCmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+info.GoArch, "CGO_ENABLED=0")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cross-compile failed: %s: %w", string(out), err)
	}

	fmt.Printf("Cross-compiled: %s (arch=%s)\n", tmpBin, info.GoArch)

	// Remove old binary and deploy new one
	s.RunCommand(fmt.Sprintf("rm -f $(eval echo %s)", remotePath))
	return s.SCPFile(tmpBin, remotePath)
}

// SCPFile copies a local file to the remote path.
// Uses cat-based transfer instead of scp protocol for reliability.
func (s *SSHSession) SCPFile(localPath, remotePath string) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer localFile.Close()

	// Resolve ~ and ensure directory exists on remote
	// Use eval to expand tilde in shell
	resolvedOut, _ := s.RunCommand(fmt.Sprintf("eval echo %s", remotePath))
	resolved := strings.TrimSpace(resolvedOut)
	if resolved == "" {
		resolved = remotePath
	}

	s.RunCommand(fmt.Sprintf("mkdir -p $(dirname %s)", resolved))

	session, err := s.Client.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	// Use cat > file approach — more reliable than scp -t with tilde paths
	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		io.Copy(w, localFile)
	}()

	cmd := fmt.Sprintf("cat > %s && chmod 755 %s", resolved, resolved)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("transfer: %w", err)
	}

	return nil
}

// findProjectRoot walks up from dir looking for go.mod.
func findProjectRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// FreeRemotePort returns a TCP port in [low, high] not currently listening on
// the remote, avoiding collisions with services already running there.
func (s *SSHSession) FreeRemotePort(low, high int) (int, error) {
	if low >= high {
		return 0, fmt.Errorf("invalid port range %d-%d", low, high)
	}
	// Collect ports already in LISTEN state (ss, falling back to netstat).
	out, _ := s.RunCommand("ss -ltnH 2>/dev/null | awk '{print $4}' | sed 's/.*://' ; netstat -ltn 2>/dev/null | awk '{print $4}' | sed 's/.*://'")
	used := map[int]bool{}
	for _, line := range strings.Split(out, "\n") {
		if p, err := strconv.Atoi(strings.TrimSpace(line)); err == nil {
			used[p] = true
		}
	}
	for i := 0; i < 50; i++ {
		p := low + rand.Intn(high-low+1)
		if !used[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no free port found in %d-%d", low, high)
}

// StopRemoteAgent kills the agent (via its pidfile) and deletes the deployed
// binary, freeing the port. Best-effort; errors are ignored.
func (s *SSHSession) StopRemoteAgent(baseDir string) {
	if baseDir == "" {
		baseDir = "~/postgresmonitor"
	}
	cmd := fmt.Sprintf(`d=$(eval echo %s); `+
		`if [ -f "$d/agent.pid" ]; then kill "$(cat "$d/agent.pid")" 2>/dev/null || true; fi; `+
		`rm -f "$d/pgmonitor" "$d/agent.pid" 2>/dev/null || true; echo cleaned`, baseDir)
	s.RunCommand(cmd)
}

// StartRemoteAgent starts the monitor agent on the remote machine.
func (s *SSHSession) StartRemoteAgent(port int, baseDir string) error {
	pidFile := baseDir + "/agent.pid"

	cmd := fmt.Sprintf("nohup %s/pgmonitor --agent --port %d --dir %s > %s/agent.log 2>&1 & echo $! > %s",
		baseDir, port, baseDir, baseDir, pidFile)
	if _, err := s.RunCommand(cmd); err != nil {
		return fmt.Errorf("start remote agent: %w", err)
	}

	// Brief wait then verify — check the response is actually our agent
	// (contains a known field) so a port-squatter can't pass as healthy.
	time.Sleep(time.Second)
	verifyCmd := fmt.Sprintf("curl -s localhost:%d/monitor 2>/dev/null", port)
	out, err := s.RunCommand(verifyCmd)
	if err != nil || !strings.Contains(out, "collectedAt") {
		logOut, _ := s.RunCommand(fmt.Sprintf("cat %s/agent.log 2>/dev/null | tail -20", baseDir))
		return fmt.Errorf("remote agent failed to start:\n%s", logOut)
	}

	fmt.Printf("Remote agent started on port %d (dir: %s)\n", port, baseDir)
	return nil
}

// ForwardPort sets up local port forwarding: local random port -> remote port.
// Returns the local port number.
func (s *SSHSession) ForwardPort(remotePort int) (int, error) {
	// Listen on random local port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("listen: %w", err)
	}

	localPort := listener.Addr().(*net.TCPAddr).Port
	s.LocalPort = localPort

	go func() {
		defer listener.Close()
		for {
			local, err := listener.Accept()
			if err != nil {
				select {
				case <-s.forwardDone:
					return
				default:
					continue
				}
			}

			remoteAddr := fmt.Sprintf("localhost:%d", remotePort)
			remote, err := s.Client.Dial("tcp", remoteAddr)
			if err != nil {
				local.Close()
				continue
			}

			go func() {
				defer local.Close()
				defer remote.Close()
				done := make(chan struct{}, 2)
				go func() { io.Copy(remote, local); done <- struct{}{} }()
				go func() { io.Copy(local, remote); done <- struct{}{} }()
				<-done
			}()
		}
	}()

	fmt.Printf("Port forward: localhost:%d -> remote:%d\n", localPort, remotePort)
	return localPort, nil
}

// Close terminates the SSH session.
func (s *SSHSession) Close() error {
	close(s.forwardDone)
	return s.Client.Close()
}

func loadPrivateKey(keyPath string) (ssh.Signer, error) {
	if keyPath == "" {
		// Try default key locations
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home: %w", err)
		}
		candidates := []string{
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
		}
		for _, c := range candidates {
			if key, err := readKey(c); err == nil {
				return key, nil
			}
		}
		return nil, fmt.Errorf("no SSH key found (tried ~/.ssh/id_ed25519, ~/.ssh/id_rsa)")
	}

	// Expand ~ in path
	if strings.HasPrefix(keyPath, "~/") {
		home, _ := os.UserHomeDir()
		keyPath = filepath.Join(home, keyPath[2:])
	}

	return readKey(keyPath)
}

func readKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ssh.ParsePrivateKey(data)
}

// resolveSSHAlias parses ~/.ssh/config to resolve a Host alias.
// Returns partial SSHConfig with whatever fields are found.
func resolveSSHAlias(alias string) SSHConfig {
	result := SSHConfig{Host: alias} // default: alias is the hostname

	home, err := os.UserHomeDir()
	if err != nil {
		return result
	}

	data, err := os.ReadFile(filepath.Join(home, ".ssh", "config"))
	if err != nil {
		return result
	}

	lines := strings.Split(string(data), "\n")
	inBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first whitespace or =
		var key, val string
		if idx := strings.IndexAny(line, " \t="); idx > 0 {
			key = strings.TrimSpace(line[:idx])
			val = strings.TrimSpace(strings.TrimLeft(line[idx:], " \t="))
		} else {
			continue
		}

		if strings.EqualFold(key, "Host") {
			// Check if this block matches our alias
			// Host can have multiple patterns separated by spaces
			patterns := strings.Fields(val)
			inBlock = false
			for _, p := range patterns {
				if p == alias {
					inBlock = true
					break
				}
			}
			continue
		}

		if !inBlock {
			continue
		}

		switch strings.ToLower(key) {
		case "hostname":
			result.Host = val
		case "user":
			result.User = val
		case "port":
			if p, err := strconv.Atoi(val); err == nil {
				result.Port = p
			}
		case "identityfile":
			// Expand ~
			if strings.HasPrefix(val, "~/") {
				val = filepath.Join(home, val[2:])
			}
			result.KeyPath = val
		}
	}

	return result
}
