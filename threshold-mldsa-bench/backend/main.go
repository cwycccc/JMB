package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"threshold-mldsa/backend/api"
	"threshold-mldsa/backend/signer"
	transactionstore "threshold-mldsa/backend/store"
	"threshold-mldsa/network"
)

func main() {
	address := os.Getenv("JMB_BACKEND_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}
	mode := os.Getenv("JMB_SIGNER_MODE")
	if mode == "" {
		mode = "distributed"
	}
	signTimeout, err := durationSetting("JMB_SIGN_TIMEOUT", 20*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	nodeTimeout, err := durationSetting("JMB_NODE_TIMEOUT", 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	service, verifier, closeService, err := signingComponents(mode, nodeTimeout)
	if err != nil {
		log.Fatal(err)
	}
	defer closeService()
	handler, err := api.NewHandler(transactionstore.New(), service, verifier, api.Options{Mode: mode, SignTimeout: signTimeout})
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      signTimeout + 10*time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("clearing backend listening on %s; signing_mode=%s; key_id=%s", address, mode, verifier.KeyID())
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func durationSetting(name string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return duration, nil
}

func signingComponents(mode string, nodeTimeout time.Duration) (signer.Service, signer.Verifier, func(), error) {
	if mode == "local" {
		local, err := signer.NewLocalSigner()
		if err != nil {
			return nil, nil, nil, err
		}
		public := local.PublicDescriptor()
		verifier, err := signer.NewVerifier(public, public.KeyID)
		return local, verifier, func() {}, err
	}
	if mode != "distributed" {
		return nil, nil, nil, fmt.Errorf("JMB_SIGNER_MODE must be distributed or local")
	}
	path := os.Getenv("JMB_CLUSTER_CONFIG")
	if path == "" {
		return nil, nil, nil, fmt.Errorf("set JMB_CLUSTER_CONFIG to the launcher's public cluster JSON file")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open public cluster configuration: %w", err)
	}
	defer file.Close()
	cluster, err := network.ReadClusterConfig(file)
	if err != nil {
		return nil, nil, nil, err
	}
	service, err := signer.NewDistributedSigner(cluster, os.Getenv(network.CoordinatorTokenEnv), nodeTimeout)
	if err != nil {
		return nil, nil, nil, err
	}
	// Construct an independent public-key verifier, without consulting a result's
	// Verified flag or generating any new key shares in the distributed backend.
	verifier, err := signer.NewVerifier(cluster.PublicConfig, cluster.KeyID)
	if err != nil {
		service.Close()
		return nil, nil, nil, err
	}
	return service, verifier, service.Close, nil
}
