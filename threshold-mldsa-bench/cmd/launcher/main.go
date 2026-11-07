// The trusted launcher generates one in-memory key set and sends exactly one
// packed share to each signer over a private anonymous stdin pipe.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
	signernode "threshold-mldsa/signer-node"
)

type node = network.ClusterNode

type child struct {
	command *exec.Cmd
	done    chan error
}

func main() {
	name := "signer"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	signer := flag.String("signer", filepath.Join("bin", name), "Path to a prebuilt signer executable")
	basePort := flag.Int("base-port", 9100, "First of five consecutive localhost ports")
	startupTimeout := flag.Duration("startup-timeout", 15*time.Second, "Maximum initialization time")
	checkOnly := flag.Bool("check-only", false, "Verify five initialized processes and stop them before exiting")
	flag.Parse()
	if err := launch(*signer, *basePort, *startupTimeout, *checkOnly); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func launch(signer string, basePort int, timeout time.Duration, checkOnly bool) error {
	token := os.Getenv(network.CoordinatorTokenEnv)
	if token == "" && checkOnly {
		var credential [32]byte
		if _, err := rand.Read(credential[:]); err != nil {
			return errors.New("credential generation failed")
		}
		token = hex.EncodeToString(credential[:])
	}
	if network.ValidateToken(token) != nil {
		return errors.New("set JMB_COORDINATOR_TOKEN to a random 64-character hexadecimal credential before launching")
	}
	if basePort < 1 || basePort > 65531 || timeout <= 0 {
		return errors.New("invalid base port or startup timeout")
	}
	signer, err := filepath.Abs(signer)
	if err != nil {
		return err
	}
	if info, err := os.Stat(signer); err != nil || info.IsDir() {
		return errors.New("signer executable not found; first run go build -o bin/signer.exe ./cmd/signer")
	}
	// Fail before generating or distributing secrets if an existing demo
	// already occupies one of these ports. Existing processes are never killed.
	for id := 0; id < 5; id++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", basePort+id))
		if err != nil {
			return fmt.Errorf("party %d port %d unavailable", id, basePort+id)
		}
		listener.Close()
	}

	config, shares, err := cryptoengine.TrustedKeySetup(3, 5)
	if err != nil {
		return errors.New("trusted key initialization failed")
	}
	defer func() {
		for _, share := range shares {
			share.Destroy()
		}
	}()
	children := make([]child, 0, 5)
	keepRunning := false
	defer func() {
		if !keepRunning {
			for _, process := range children {
				_ = process.command.Process.Kill()
			}
			for _, process := range children {
				<-process.done
			}
		}
	}()

	nodes := make([]node, 0, 5)
	for id := 0; id < 5; id++ {
		data, err := cryptoengine.ExportPartyShare(config, shares[id])
		if err != nil {
			return errors.New("share export failed")
		}
		initialization, err := signernode.EncodeInitialization(data, token)
		clear(data)
		if err != nil {
			return errors.New("node initialization encoding failed")
		}
		data = initialization
		command := exec.Command(signer, fmt.Sprintf("-id=%d", id),
			fmt.Sprintf("-port=%d", basePort+id), "-init-stdin", "-key-id="+config.KeyID())
		// The node receives its credential only through its private pipe.
		for _, variable := range os.Environ() {
			name, _, _ := strings.Cut(variable, "=")
			if !strings.EqualFold(name, network.CoordinatorTokenEnv) {
				command.Env = append(command.Env, variable)
			}
		}
		// No secrets in argv/environment, log files or inherited descriptors.
		// Child stdout/stderr go to the null device. Health checks report startup
		// failure without echoing potentially sensitive child diagnostics.
		pipe, err := command.StdinPipe()
		if err != nil {
			clear(data)
			return errors.New("initialization pipe creation failed")
		}
		if err := command.Start(); err != nil {
			pipe.Close()
			clear(data)
			return fmt.Errorf("party %d process start failed", id)
		}
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		children = append(children, child{command: command, done: done})
		_, writeErr := pipe.Write(data)
		clear(data)
		closeErr := pipe.Close()
		if writeErr != nil || closeErr != nil {
			return fmt.Errorf("party %d initialization delivery failed", id)
		}
		nodes = append(nodes, node{PartyID: id, PID: command.Process.Pid, Port: basePort + id, Address: fmt.Sprintf("http://127.0.0.1:%d", basePort+id)})
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	for _, node := range nodes {
		if err := waitReady(ctx, client, node, config.KeyID()); err != nil {
			return err
		}
	}
	for _, process := range children {
		select {
		case err := <-process.done:
			process.done <- err // Leave the result available for deferred cleanup.
			return errors.New("a signer exited during initialization")
		default:
		}
	}

	status := "CRYPTO_READY"
	if checkOnly {
		status = "CHECK_PASSED"
	}
	if err := json.NewEncoder(os.Stdout).Encode(network.ClusterConfig{
		Version: 1, Status: status, KeyID: config.KeyID(), PublicConfig: config.PublicDescriptor(), Nodes: nodes,
	}); err != nil {
		return err
	}
	keepRunning = !checkOnly
	// Normal return leaves the five child processes alive and releases all
	// initializer references. Go does not guarantee erasure of heap copies.
	return nil
}

func waitReady(ctx context.Context, client *http.Client, node node, keyID string) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("http://127.0.0.1:%d/health", node.Port), nil)
		if err != nil {
			return err
		}
		response, err := client.Do(request)
		if err == nil {
			var health struct {
				PartyID   int    `json:"party_id"`
				Status    string `json:"status"`
				KeyLoaded bool   `json:"key_loaded"`
				KeyID     string `json:"key_id"`
				Threshold int    `json:"threshold"`
				Total     int    `json:"total_parties"`
			}
			decodeErr := json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&health)
			response.Body.Close()
			if response.StatusCode == http.StatusOK && decodeErr == nil {
				if health.PartyID != node.PartyID || health.KeyID != keyID || !health.KeyLoaded ||
					health.Status != "CRYPTO_READY" || health.Threshold != 3 || health.Total != 5 {
					return fmt.Errorf("party %d reported an unexpected initialization identity", node.PartyID)
				}
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("party %d initialization timed out", node.PartyID)
		case <-ticker.C:
		}
	}
}
