# C2.2: trusted local key launcher

Run these commands in PowerShell from `threshold-mldsa-bench` (Go 1.24+):

```powershell
$env:Path = "C:\Users\Lenovo\Desktop\金密杯\.tools\go\bin;$env:Path"
$env:GOTOOLCHAIN = "local"
go test ./... -count=1
go build -o ./bin/signer.exe ./cmd/signer
go build -o ./bin/launcher.exe ./cmd/launcher
```

For C2.3, set `JMB_COORDINATOR_TOKEN` to a random 64-character hexadecimal
credential before a normal launch. See [the complete PowerShell instructions](../../network/README.md).
`-check-only` can generate an ephemeral credential internally. The trusted pipe
now wraps the unchanged C2.2 share envelope together with this credential.

After setting that credential, launch with `.\bin\launcher.exe -base-port=9100`.

Stop the old NETWORK_READY demo nodes with Ctrl+C in their original terminals
before launching. The launcher refuses occupied ports and does not stop existing
processes. To run alongside them, use `-base-port=9200` instead.

On success the launcher prints public JSON containing one KeyID and the five
Party IDs, process IDs, and ports, then exits. The five signer processes continue
running. Inspect them with:

```powershell
0..4 | ForEach-Object { Invoke-RestMethod "http://127.0.0.1:910$($_)/health" }
```

Every health response must report CRYPTO_READY, key_loaded=true, threshold=3,
total_parties=5, and the same key_id. The launcher has already checked these.
Capture the launcher's public process list if you want convenient cleanup:

```powershell
$cluster = .\bin\launcher.exe -base-port=9200 | ConvertFrom-Json
$cluster.nodes | Format-Table
# After the demonstration, stop only these specific child processes:
$cluster.nodes | ForEach-Object { Stop-Process -Id $_.pid }
```

For automatic acceptance without leaving nodes running:

```powershell
.\bin\launcher.exe -base-port=9300 -check-only
```

The launcher exports exactly one threshold-native share per child via a separate
anonymous stdin pipe. No private share is put in argv, environment variables,
files, HTTP, or logs. The coordinator credential is supplied by the launcher's
environment and used in HTTP Authorization headers; it is not printed or logged.
Public KeyID commits to the system public key, parameters, and
the canonical hashes of all per-node shares. Each signer independently validates
its expected PartyID and the separately supplied expected KeyID before listening.
The adapter bounds the envelope and validates the private Pack layout (including
share masks and eta coefficient ranges) before invoking the unchecked native
Unpack method. This is not ordinary ML-DSA private-key encoding.

Export invalidates the original in-memory share handle. Import creates a fresh
process-local setup identity and an empty session map. Encoded byte buffers are
cleared after use and remaining initialization handles are released; Go cannot
guarantee erasure of all heap copies. A copied plaintext envelope is not globally
single-use: its confidentiality relies on the trusted launcher and pipe. Never
persist or transmit this format through an untrusted channel.

Restart the entire demo with a fresh launcher/KeyID if a signer exits. Do not
restore old signing state. C2.3 adds authenticated network Round1/2/3 transport;
see the network README. The financial backend remains on its earlier local API.
