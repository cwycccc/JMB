# C2.4：金融清算后端调用网络门限签名

默认签名模式是 `distributed`。后端只加载 Launcher 输出的公共配置和协调器凭据，
使用 P0/P2/P4 的独立节点执行三轮 HTTP 协议，再用公共配置中的系统公钥独立验签。
三个金融 API 的路径保持不变。`local` 是需要显式选择的回归基线，不会在网络失败时自动启用。

## PowerShell 完整复现

先关闭旧版本的后端和 signer，再从项目根目录执行。Windows 下运行中的 exe
可能无法被重新编译覆盖。以下使用 9400–9404 作为节点端口，后端使用 8080。

```powershell
cd "C:\Users\Lenovo\Desktop\金密杯\Threshold-ML-DSA-main\threshold-mldsa-bench"
$env:Path = "C:\Users\Lenovo\Desktop\金密杯\.tools\go\bin;$env:Path"
$env:GOTOOLCHAIN = "local"
go build -o ./bin/signer.exe ./cmd/signer
go build -o ./bin/launcher.exe ./cmd/launcher
go build -o ./bin/backend.exe ./backend

# 生成协调器口令，不打印、不写入公共配置。
$credential = New-Object byte[] 32
$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$rng.GetBytes($credential)
$rng.Dispose()
$env:JMB_COORDINATOR_TOKEN = -join ($credential | ForEach-Object { $_.ToString("x2") })
[Array]::Clear($credential, 0, $credential.Length)

$clusterJson = .\bin\launcher.exe -base-port=9400
if ($LASTEXITCODE -ne 0) { throw "Launcher failed" }
$clusterJson | Set-Content -Encoding utf8 ./cluster.public.json
$cluster = $clusterJson | ConvertFrom-Json
$cluster.nodes | Format-Table

$env:JMB_CLUSTER_CONFIG = (Resolve-Path ./cluster.public.json).Path
$env:JMB_SIGNER_MODE = "distributed"
.\bin\backend.exe
```

启动日志应包含 `signing_mode=distributed`，其 `key_id` 与 `$cluster.key_id` 相同。
后端进程直接持有网络 Coordinator，无需另外启动协调器 HTTP 服务。
`cluster.public.json` 只含版本、KeyID、系统公钥、公共份额摘要和节点地址/PID，
不含私钥份额、初始化种子或协调器口令；读取器支持 Windows UTF-8 BOM。

保持第一个窗口运行，在第二个 PowerShell 窗口执行：

```powershell
$transaction = @{
    version = "clearing-v1"
    transaction_id = "TX001"
    from_bank = "BankA"
    to_bank = "BankB"
    amount_fen = 100000000
    currency = "CNY"
    nonce = "nonce-001"
} | ConvertTo-Json

Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8080/api/transactions" -ContentType "application/json" -Body $transaction

$result = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:8080/api/transactions/TX001/sign"
$result | Format-List

Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/transactions/TX001" | ConvertTo-Json -Depth 8
```

预期字段：`participants=[0,2,4]`、`signing_mode=distributed`、`status=VERIFIED`、
`verified=true`，以及同一系统的 `key_id`、唯一 `session_id`、真实签名与耗时。
`amount_fen=100000000` 表示 1,000,000.00 CNY。

在第一个窗口按 Ctrl+C 关闭后端，然后关闭本次启动的节点：

```powershell
$cluster.nodes | ForEach-Object { Stop-Process -Id $_.pid }
Remove-Item Env:JMB_COORDINATOR_TOKEN
```

## 状态、指标与失败处理

- 创建交易返回 `PENDING`。开始签名前，内存 Store 原子地切换到 `SIGNING`。
  同一交易的并发签名、重复创建和已签名交易再次签名都返回 409。
- 后端将规范消息的副本交给签名服务，随后用保留的原始规范字节、公钥和返回签名
  独立验签；返回的 KeyID 不匹配或验签失败时记录 `FAILED`，不保留无效签名。
- `duration_ms` 是签名调用加后端独立验签的实际耗时；`signing_duration_ms` 包含
  节点可用性检查和网络协议；`verification_duration_ms` 是后端独立验签耗时。
- 分布式 `attempts` 从 1 开始计数，对应协议的 `Attempt+1`；
  `round_message_counts` 是成功签名过程中所有尝试的实际响应计数。
  失败时没有完整的轮次计数就省略该字段，不伪造为零次网络交互。
- 节点不足或配置不一致返回 503/FAILED。超时返回 504/FAILED。
  服务不会自动回退到本地签名，也不会自动重发密码轮次 POST。
- 超时后仅在 Abort 已确认时允许业务重试；该重试会创建新的 SessionID。
  `abort_confirmed=false` 时同时设置 `requires_reinitialization=true`，同一交易
  重试返回 409，当前适配器的后续签名返回 503，必须整体重新初始化节点和后端。
- 配置项：`JMB_CLUSTER_CONFIG` 为公共配置路径；`JMB_COORDINATOR_TOKEN` 为独立凭据；
  `JMB_NODE_TIMEOUT` 默认 `3s`；`JMB_SIGN_TIMEOUT` 默认 `20s`；
  `JMB_BACKEND_ADDR` 默认 `127.0.0.1:8080`。
- 显式设置 `JMB_SIGNER_MODE=local` 可运行本地基线。该模式创建本地密钥，
  不要求集群配置。旧本地接口不提供实际尝试次数，因此 `attempts=0` 表示未报告，
  不是零次执行，且不报告网络会话或网络轮次计数。

交易和幂等状态仍只保存在内存中。重启后端不会保留交易记录；更换 KeyID 时应
重新启动后端以加载新公共配置。跨重启历史交易幂等和密钥持久化不在本阶段范围内。

## 自动验收

```powershell
go test ./... -count=1
go vet ./...

# 显式启用真实节点进程的金融 API 集成测试。
$env:C24_SIGNER_BINARY = (Resolve-Path ./bin/signer.exe).Path
$env:C24_LAUNCHER_BINARY = (Resolve-Path ./bin/launcher.exe).Path
$env:C24_BACKEND_BINARY = (Resolve-Path ./bin/backend.exe).Path
go test -p 1 ./backend/... -v -count=1
```

真实进程测试自动选择空闲端口并清理自己创建的进程。未设置相应二进制路径时，
真实进程用例会显示 SKIP；SKIP 不能作为 C2.4 分布式验收通过的证据。
