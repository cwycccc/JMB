package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"threshold-mldsa/cryptoengine"
	"threshold-mldsa/network"
	signernode "threshold-mldsa/signer-node"
)

func main() {
	id := flag.Int("id", 0, "Party ID")
	port := flag.Int("port", 9100, "HTTP port")
	initStdin := flag.Bool("init-stdin", false, "Load this Party's share from a dedicated stdin pipe")
	expectedKeyID := flag.String("key-id", "", "Expected public KeyID supplied by the trusted launcher")
	flag.Parse()

	if *id < 0 || *id >= 5 {
		log.Fatal("Party ID must be 0-4")
	}
	if *port < 1 || *port > 65535 {
		log.Fatal("HTTP port must be 1-65535")
	}
	var config *cryptoengine.PublicConfig
	var party cryptoengine.Party
	var coordinatorToken string
	if *initStdin {
		var err error
		config, party, coordinatorToken, err = signernode.ReadInitialization(os.Stdin, *id, *expectedKeyID)
		_ = os.Stdin.Close()
		if err != nil {
			log.Fatal("share initialization rejected")
		}
	} else if *expectedKeyID != "" {
		log.Fatal("-key-id requires -init-stdin")
	}

	var handler http.Handler
	if party != nil {
		var err error
		handler, err = signernode.NewHandler(config, party, coordinatorToken)
		if err != nil {
			log.Fatal("node transport initialization failed")
		}
	} else {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(network.Health{PartyID: *id, Status: "NETWORK_READY", KeyLoaded: false})
		})
		handler = mux
	}

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	log.Printf("Party %d listening on %s", *id, addr)
	server, err := signernode.NewServer(addr, handler)
	if err != nil {
		log.Fatal("invalid node address")
	}
	log.Fatal(server.ListenAndServe())
}
