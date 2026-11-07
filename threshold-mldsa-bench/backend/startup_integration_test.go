package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"threshold-mldsa/backend/clearing"
	"threshold-mldsa/backend/signer"
	"threshold-mldsa/network"
)

func TestBackendExecutableUsesLauncherPublicConfiguration(t *testing.T) {
	launcher, signerPath, backend := os.Getenv("C24_LAUNCHER_BINARY"), os.Getenv("C24_SIGNER_BINARY"), os.Getenv("C24_BACKEND_BINARY")
	if launcher == "" || signerPath == "" || backend == "" {
		t.Skip("set C24_LAUNCHER_BINARY, C24_SIGNER_BINARY and C24_BACKEND_BINARY for executable startup acceptance")
	}
	var credential [32]byte
	if _, err := rand.Read(credential[:]); err != nil {
		t.Fatal(err)
	}
	token := hex.EncodeToString(credential[:])
	clear(credential[:])
	base := startupPorts(t)
	launch := exec.Command(launcher, "-signer="+signerPath, fmt.Sprintf("-base-port=%d", base))
	launch.Env = append(os.Environ(), network.CoordinatorTokenEnv+"="+token)
	publicJSON, err := launch.Output()
	if err != nil {
		t.Fatal("launcher failed")
	}
	cluster, err := network.ReadClusterConfig(bytes.NewReader(publicJSON))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, node := range cluster.Nodes {
			if process, err := os.FindProcess(node.PID); err == nil {
				_ = process.Kill()
				_, _ = process.Wait()
				_ = process.Release()
			}
		}
	})
	// Generated test artifact contains public information only; exercise the
	// actual file loader including Windows PowerShell's optional UTF-8 BOM.
	path := filepath.Join(t.TempDir(), "cluster.public.json")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, publicJSON...), 0600); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()
	command := exec.Command(backend)
	command.Env = append(os.Environ(), "JMB_SIGNER_MODE=distributed", "JMB_BACKEND_ADDR="+address,
		"JMB_CLUSTER_CONFIG="+path, network.CoordinatorTokenEnv+"="+token)
	if err := command.Start(); err != nil {
		t.Fatal("backend executable did not start")
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	t.Cleanup(func() { _ = command.Process.Kill(); <-done })
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{Proxy: nil}}
	t.Cleanup(client.CloseIdleConnections)
	ready := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		response, err := client.Get("http://" + address + "/api/transactions/startup-probe")
		if err == nil {
			response.Body.Close()
			if response.StatusCode == 404 {
				ready = true
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !ready {
		t.Fatal("backend HTTP service did not become ready")
	}
	tx := clearing.Transaction{Version: "clearing-v1", TransactionID: "EXEC001", FromBank: "BankA", ToBank: "BankB", AmountFen: 100000000, Currency: "CNY", Nonce: "exec-nonce"}
	data, err := json.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Post("http://"+address+"/api/transactions", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 201 {
		t.Fatal("executable API could not create transaction")
	}
	response, err = client.Post("http://"+address+"/api/transactions/EXEC001/sign", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result struct {
		Status        string  `json:"status"`
		Verified      bool    `json:"verified"`
		KeyID         string  `json:"key_id"`
		Mode          string  `json:"signing_mode"`
		Signature     string  `json:"signature_base64"`
		DurationMS    float64 `json:"duration_ms"`
		Attempts      int     `json:"attempts"`
		RoundMessages [3]int  `json:"round_message_counts"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || result.Status != "VERIFIED" || !result.Verified || result.KeyID != cluster.KeyID || result.Mode != "distributed" {
		t.Fatal("backend did not use the launcher key for distributed signing")
	}
	verifier, err := signer.NewVerifier(cluster.PublicConfig, cluster.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	message, err := clearing.SigningMessage(tx)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := base64.StdEncoding.DecodeString(result.Signature)
	if err != nil {
		t.Fatal(err)
	}
	if !verifier.Verify(result.KeyID, message, signature) {
		t.Fatal("executable API signature failed independent verification")
	}
	t.Logf("real backend.exe + five signer processes: mode=%s; status=%s; launcher_key_id_matches=true; attempts=%d; rounds=%v; duration_ms=%.4f", result.Mode, result.Status, result.Attempts, result.RoundMessages, result.DurationMS)
}

func startupPorts(t *testing.T) int {
	t.Helper()
	for tries := 0; tries < 30; tries++ {
		first, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		base := first.Addr().(*net.TCPAddr).Port
		listeners := []net.Listener{first}
		if base <= 65531 {
			for offset := 1; offset < 5; offset++ {
				listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", base+offset))
				if err != nil {
					break
				}
				listeners = append(listeners, listener)
			}
		}
		for _, listener := range listeners {
			listener.Close()
		}
		if len(listeners) == 5 {
			return base
		}
	}
	t.Fatal("no five-port range available")
	return 0
}
