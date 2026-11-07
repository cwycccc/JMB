# C2.3: HTTP threshold ML-DSA-44

Run from `threshold-mldsa-bench` in PowerShell using Go 1.24. Stop old signer
processes first, or choose a different free range of five localhost ports.
Rebuild both signer and launcher: the stdin bootstrap now additionally carries
the coordinator credential; the C2.2 share envelope itself is unchanged.

```powershell
$env:Path = "C:\Users\Lenovo\Desktop\金密杯\.tools\go\bin;$env:Path"
$env:GOTOOLCHAIN = "local"
go build -o ./bin/signer.exe ./cmd/signer
go build -o ./bin/launcher.exe ./cmd/launcher
go build -o ./bin/network-demo.exe ./cmd/network-demo

# Generate a credential without printing it. Keep this PowerShell window open.
$credential = New-Object byte[] 32
$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$rng.GetBytes($credential)
$rng.Dispose()
$env:JMB_COORDINATOR_TOKEN = -join ($credential | ForEach-Object { $_.ToString("x2") })
[Array]::Clear($credential, 0, $credential.Length)

$cluster = .\bin\launcher.exe -base-port=9300 | ConvertFrom-Json
if ($LASTEXITCODE -ne 0) { throw "Launcher failed" }
$cluster.nodes | Format-Table

9300..9304 | ForEach-Object { Invoke-RestMethod "http://127.0.0.1:$_/health" }

$result = .\bin\network-demo.exe -base-port=9300 -key-id $cluster.key_id | ConvertFrom-Json
$result | Select-Object session_id, attempt, participants, verified, tampered_verified, duration_ms, round_message_counts | Format-List
```

The default participants are P0/P2/P4. The signed message is
`TX001|BankA|BankB|1000000.00|CNY`; the altered-amount verification uses
`TX001|BankA|BankB|2000000.00|CNY`. Expect `verified=true` and
`tampered_verified=false`. Round message counts are `[3,3,3]` if the first
attempt combines successfully, or multiples of three after fresh attempts.
Attempt numbering starts at zero. The full JSON includes the real Base64
signature and measured wall-clock signing duration in milliseconds.

Custom messages and participant lists:

```powershell
.\bin\network-demo.exe -base-port=9300 -key-id $cluster.key_id -participants "0,2,4" -message "TX002|BankA|BankB|3000000.00|CNY" -tampered-message "TX002|BankA|BankB|4000000.00|CNY"
```

Stop only the processes from this launch after the experiment:

```powershell
$cluster.nodes | ForEach-Object { Stop-Process -Id $_.pid }
Remove-Item Env:JMB_COORDINATOR_TOKEN
```

## Protocol

- `/health` exposes public readiness only. `GET /internal/config` exposes the
  public key and public share commitments to an authenticated coordinator.
- POST `/internal/round1`, `/internal/round2`, `/internal/round3`,
  `/internal/status`, and `/internal/abort` require a loopback source and a
  bearer credential. The launcher sends this credential on each child's private
  stdin pipe and removes it from the child's inherited environment. No
  credential, private share, seed, or private signing state is logged.
- Every request and public response binds wire version, SessionID, Attempt,
  KeyID, strictly ascending participant list, and the SHA-256 hash of the exact
  original message bytes. Round2 also supplies those bytes. The node checks its
  own relayed commitment in addition to the underlying protocol checks.
- `wirecodec.go` accepts public protocol messages only. It checks versions,
  exact round sizes, source IDs and ordering before cryptographic processing.
  `sharecodec.go` and `keysetup.go` were not changed for C2.3.
- The network coordinator owns a validated public configuration only. It sends
  one HTTP request per Party per round, collects actual responses, and calls
  `cryptoengine.Coordinator.CombineAndVerify`. It never calls centralized
  `cryptoengine.SignMessage`.
- Combine rejection triggers an authenticated Abort on every participant.
  Only after all acknowledgments does the coordinator increment Attempt and
  restart Round1 with fresh private randomness.
- A timeout or response-binding failure is never retried as a cryptographic
  POST. The coordinator queries state and sends Abort using a separate cleanup
  deadline. It then returns an `AttemptError`, without silently resuming from an
  uncertain result. If Abort cannot be confirmed, stop and reinitialize the
  whole demo before further work.
- Nodes retain terminal tombstones for this controlled launch lifecycle. A new
  attempt cannot overtake an active one; old attempts cannot be revived. The
  in-memory record limit is 4096 per node; new work is refused at capacity
  rather than discarding replay markers.
- There is no cross-restart replay persistence. Restart the entire environment
  with a new KeyID after a node exits. Historical transaction idempotency still
  belongs in the business backend. This milestone does not change that backend.

## Acceptance tests

Regular codec, binding, authentication and prior regression tests:

```powershell
go test ./... -count=1
go vet ./...
```

The five-process test is explicitly enabled with the rebuilt executables. It
chooses free ports, launches five real signer processes, and kills only those
test processes afterwards. It also checks transcript-only recombination,
modified-amount rejection, unauthenticated requests, an injected Combine
rejection, an executed Round2 whose HTTP response is lost, and unconfirmed Abort.

```powershell
$env:C23_SIGNER_BINARY = (Resolve-Path ./bin/signer.exe).Path
$env:C23_LAUNCHER_BINARY = (Resolve-Path ./bin/launcher.exe).Path
go test ./network -run TestNetworkFiveProcesses -v -count=1
```

Without those two environment variables, the real-process test reports SKIP;
that is not evidence that cross-process acceptance has passed.
