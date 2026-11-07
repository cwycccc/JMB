package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"threshold-mldsa/network"
)

func main() {
	basePort := flag.Int("base-port", 9200, "First signer port")
	keyID := flag.String("key-id", "", "Expected KeyID printed by launcher")
	participantsFlag := flag.String("participants", "0,2,4", "Comma-separated Party IDs")
	message := flag.String("message", "TX001|BankA|BankB|1000000.00|CNY", "Exact message bytes to sign")
	tampered := flag.String("tampered-message", "TX001|BankA|BankB|2000000.00|CNY", "Altered transaction to reject")
	flag.Parse()
	if *basePort < 1 || *basePort > 65531 || len(*keyID) != 64 || *message == *tampered {
		fail("invalid port, KeyID or tampered message")
	}
	var participants []int
	for _, text := range strings.Split(*participantsFlag, ",") {
		id, err := strconv.Atoi(text)
		if err != nil {
			fail("invalid participant list")
		}
		participants = append(participants, id)
	}
	endpoints := make(map[int]string)
	for id := 0; id < 5; id++ {
		endpoints[id] = fmt.Sprintf("http://127.0.0.1:%d", *basePort+id)
	}
	client, err := network.NewClient(endpoints, os.Getenv(network.CoordinatorTokenEnv), 5*time.Second)
	if err != nil {
		fail(err.Error())
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	coordinator, err := network.Connect(ctx, client, participants, *keyID)
	if err != nil {
		fail(err.Error())
	}
	result, err := coordinator.Sign(ctx, []byte(*message), participants)
	if err != nil {
		fail(err.Error())
	}
	modifiedVerified := coordinator.Verify([]byte(*tampered), result.Signature)
	if !coordinator.Verify([]byte(*message), result.Signature) || modifiedVerified {
		fail("signature acceptance check failed")
	}
	_ = json.NewEncoder(os.Stdout).Encode(struct {
		*network.Result
		TamperedVerified bool `json:"tampered_verified"`
	}{Result: result, TamperedVerified: modifiedVerified})
}

func fail(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
