package ssh

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/bascanada/logviewer/pkg/adapter/hl"
	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/reader"
	sshc "golang.org/x/crypto/ssh"
	"k8s.io/client-go/util/homedir"
)

const (
	OptionsCmd = "cmd"
	// OptionsPaths specifies file paths to read logs from on the remote host.
	// When paths are provided, a hybrid command will be used that checks for hl on the remote host.
	OptionsPaths = "paths"
	// OptionsPreferNativeDriver when set to true, disables hl usage and forces the native command.
	OptionsPreferNativeDriver = "preferNativeDriver"
)

type SSHLogClientOptions struct {
	User string `json:"user"`
	Addr string `json:"addr"`

	PrivateKey string `json:"privateKey"`
	DisablePTY bool   `json:"disablePTY"`
}

type sshLogClient struct {
	conn    *sshc.Client
	options SSHLogClientOptions
}

func getCommand(search *client.LogSearch) (string, error) {
	cmdTplStr := search.Options.GetString(OptionsCmd)

	if cmdTplStr == "" {
		return "", errors.New("cmd is missing for sshLogClient")
	}

	tmpl, err := template.New("cmd").Parse(cmdTplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse command template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, search); err != nil {
		return "", fmt.Errorf("failed to execute command template: %w", err)
	}
	return buf.String(), nil
}

func (lc sshLogClient) Get(ctx context.Context, search *client.LogSearch) (client.LogSearchResult, error) {
	// Check if we should use hl with paths
	paths, hasPaths := search.Options.GetListOfStringsOk(OptionsPaths)
	preferNative := search.Options.GetBool(OptionsPreferNativeDriver)

	var cmd string
	var useHybridHL bool

	if hasPaths && len(paths) > 0 && !preferNative {
		// Build hybrid command that checks for hl on remote host
		hybridCmd, err := lc.buildHybridHLCommand(search, paths)
		if err != nil {
			slog.Warn("failed to build hybrid hl command, falling back to native", "error", err)
		} else {
			cmd = hybridCmd
			useHybridHL = true
			slog.Debug("using hybrid hl command for SSH", "cmd", cmd)
		}
	}

	// Fall back to native command if hybrid didn't work
	if cmd == "" {
		var err error
		cmd, err = getCommand(search)
		if err != nil {
			if err.Error() == "cmd is missing for sshLogClient" {
				// Check if we have paths but failed to build hybrid command
				if hasPaths && len(paths) > 0 {
					return nil, errors.New("failed to build hl command and no fallback 'cmd' is configured")
				}
				return nil, fmt.Errorf("configuration error: %w", err)
			}
			return nil, err
		}
		slog.Debug("using native command for SSH", "cmd", cmd)
	}

	session, err := lc.conn.NewSession()
	if err != nil {
		return nil, err
	}

	modes := sshc.TerminalModes{
		sshc.ECHO:          0,     // disable echoing
		sshc.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		sshc.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Determine whether to disable PTY, with search options overriding client options.
	disablePTY := lc.options.DisablePTY
	if searchDisable, ok := search.Options.GetBoolOk("disablePTY"); ok {
		disablePTY = searchDisable
	}

	// Only request a PTY if it's not disabled.
	if !disablePTY {
		err = session.RequestPty("xterm", 80, 40, modes)
		if err != nil {
			return nil, err
		}
	}

	_, err = session.StdinPipe()
	if err != nil {
		return nil, err
	}

	errOut, err := session.StderrPipe()
	if err != nil {
		return nil, err
	}

	out, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := session.Start(cmd); err != nil {
		return nil, fmt.Errorf("failed to start ssh command: %w", err)
	}

	// Track which engine was used (for hybrid mode)
	var engineUsed string
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		// Read stderr to detect engine marker and capture errors
		stderrScanner := bufio.NewScanner(errOut)
		var stderrOutput bytes.Buffer
		for stderrScanner.Scan() {
			line := stderrScanner.Text()
			// Check for engine marker
			if strings.HasPrefix(line, "HL_ENGINE=") {
				engineUsed = strings.TrimPrefix(line, "HL_ENGINE=")
				slog.Debug("remote engine detected", "engine", engineUsed)
				continue
			}
			stderrOutput.WriteString(line)
			stderrOutput.WriteString("\n")
		}
		if err := session.Wait(); err != nil {
			if stderrOutput.Len() > 0 {
				errChan <- fmt.Errorf("ssh command failed: %w (remote output: %s)", err, stderrOutput.String())
			} else {
				errChan <- fmt.Errorf("ssh command failed: %w", err)
			}
		}
	}()

	scanner := bufio.NewScanner(out)

	// For hybrid mode, we might get pre-filtered results
	// We'll mark it as potentially pre-filtered; the reader will handle appropriately
	searchToUse := search
	if useHybridHL {
		preFilteredSearch := *search
		if preFilteredSearch.Options == nil {
			preFilteredSearch.Options = make(map[string]interface{})
		}
		// Mark as potentially pre-filtered (if hl ran on remote, it's filtered)
		// The reader should be lenient - if filters don't match, assume pre-filtered
		preFilteredSearch.Options["__hybridHL__"] = true
		searchToUse = &preFilteredSearch
	}

	result, err := reader.GetLogResult(searchToUse, scanner, session)
	if err != nil {
		return nil, err
	}
	result.ErrChan = errChan

	// Store engine info for debugging/metrics
	_ = engineUsed // Available for future use (metrics, logging)

	return result, nil
}

// buildHybridHLCommand creates a shell command that:
// 1. Checks if hl is available on the remote host
// 2. Uses hl with filters if available (server-side filtering)
// 3. Falls back to cat/tail if hl is not available (client-side filtering)
func (lc sshLogClient) buildHybridHLCommand(search *client.LogSearch, paths []string) (string, error) {
	// Build hl arguments
	hlArgs, err := hl.BuildArgs(search, nil) // Don't include paths in args, we'll add them separately
	if err != nil {
		return "", fmt.Errorf("failed to build hl arguments: %w", err)
	}

	// Remove paths from args (they're passed separately to BuildSSHCommand)
	var argsWithoutPaths []string
	for _, arg := range hlArgs {
		isPth := false
		for _, p := range paths {
			if arg == p {
				isPth = true
				break
			}
		}
		if !isPth {
			argsWithoutPaths = append(argsWithoutPaths, arg)
		}
	}

	// Build fallback command
	var fallbackCmd string
	if fallback, err := getCommand(search); err == nil && fallback != "" {
		fallbackCmd = fallback
	} else if search.Follow {
		// For follow mode, use tail -f as fallback
		fallbackCmd = "" // hl.BuildFollowSSHCommand will handle this
	}

	// Use the SSH builder with marker for engine detection
	var cmd string
	if search.Follow {
		cmd = hl.BuildSSHCommandWithMarker(argsWithoutPaths, paths, fallbackCmd)
	} else {
		if fallbackCmd == "" {
			// Default fallback: cat the files
			var catParts []string
			catParts = append(catParts, "cat")
			for _, p := range paths {
				catParts = append(catParts, hl.ArgsToString([]string{p}))
			}
			fallbackCmd = strings.Join(catParts, " ")
		}
		cmd = hl.BuildSSHCommandWithMarker(argsWithoutPaths, paths, fallbackCmd)
	}

	return cmd, nil
}

// sessionCloser wraps an SSH session to implement io.Closer
type sessionCloser struct {
	session *sshc.Session
	stdout  io.Reader
}

func (sc *sessionCloser) Read(p []byte) (n int, err error) {
	return sc.stdout.Read(p)
}

func (sc *sessionCloser) Close() error {
	return sc.session.Close()
}

func (lc sshLogClient) GetFieldValues(ctx context.Context, search *client.LogSearch, fields []string) (map[string][]string, error) {
	// For SSH/text-based backends, we need to run a search and extract field values
	result, err := lc.Get(ctx, search)
	if err != nil {
		return nil, err
	}
	return client.GetFieldValuesFromResult(ctx, result, fields)
}

func GetLogClient(options SSHLogClientOptions) (client.LogClient, error) {

	if options.Addr == "" {
		return nil, errors.New("ssh address (addr) is empty")
	}
	if options.User == "" {
		return nil, errors.New("ssh user (user) is empty")
	}

	var privateKeyFile string
	if options.PrivateKey != "" {
		privateKeyFile = options.PrivateKey
	} else {
		privateKeyFile = filepath.Join(homedir.HomeDir(), ".ssh", "id_rsa")
	}

	key, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}
	signer, err := sshc.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	sshConfig := &sshc.ClientConfig{
		User: options.User,
		Auth: []sshc.AuthMethod{
			sshc.PublicKeys(signer),
		},
		HostKeyCallback: sshc.HostKeyCallback(
			func(hostname string, remote net.Addr, key sshc.PublicKey) error {
				return nil
			}),
	}

	conn, err := sshc.Dial("tcp", options.Addr, sshConfig)
	if err != nil {
		return nil, err
	}

	return sshLogClient{conn: conn, options: options}, nil
}
