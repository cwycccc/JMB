# C3 清算签名监控台

## 清算业务风格版（参考用户提供的视觉原型）

参考 `quantum_bank_clearing_dashboard_v2.html` 的深海军蓝、金色、金额主卡、付款行到收款行路径、机构图标和分区布局。参考文件本身未修改，其离线脚本、虚构账号、固定验签结论和“至少三个”的演示选择逻辑没有迁入本项目。

- 金额与付款/收款行在创建交易之前显示“—”；创建后只从对应的已保存交易记录或成功创建请求的字段快照读取，不随表单草稿改变。金额仍从整数分精确格式化，不经浮点乘除转换。
- 认证主卡继续由真实 API 的 VERIFIED、参与方、耗时、尝试次数驱动，明确说明密码认证不代表资金已经结算，不展示不存在的银行账号字段。
- 当前实现仍要求恰好三个参与方，节点 P0–P4 不冒充已部署的真实银行，机构名称仅为可输入的模拟交易字段。
- 实验历史快照、本页会话、协议数据缺口和安全用例均保留。新增顶部历史平均耗时也从已加载的原始实验样本重新计算，不固定为原型数值。
- 桌面双栏、手机单栏，采用本地 SVG 机构图标，不引入外部字体、CDN、前端构建工具或新的业务依赖。

此版另外编译为 `bin/backend-clearing.exe`。原先正在运行的服务没有被强行替换；要让既有一键演示程序启动新样式，请正常停止原服务后重新执行：

```powershell
go build -o ./bin/backend.exe ./backend
./bin/experiments.exe -iterations=10 -warmup=0 -port=8081 -preview=30m -output=experiments/demo-clearing
```

然后打开 `http://127.0.0.1:8081/` 并按 Ctrl+F5。系统 PATH 中没有 Go 时，使用本文后面的便携 Go 完整路径。截图和验收记录在 `experiments/clearing-style/`。

## 2026-11-07：控制台信息层级与观测优化

这次仅修改 `backend/web` 前端资源及前端测试/说明，没有修改金融 API、网络协议、秘密份额编解码或密码核心。

- 首屏优先显示 ML-DSA-44、3/5 门限、公开就绪状态、五节点到协调器的拓扑以及当前验签结果；表单移到其后。节点详情按钮与参与方勾选分开，健康检查更新不重建节点 DOM，避免丢失键盘焦点。
- 点节点可查看 PartyID、公开状态、KeyID 和已返回的参与记录。现有 API 没有逐节点消息计数，页面不会将全局总数平均分配给节点。
- 每轮可展开实际 HTTP 请求/响应字节和原始 Payload。尝试历史明确显示 API 只提供累计次数，未提供逐 Attempt 事件、拒绝原因、时间戳和逐次 Abort 确认。不会从尝试数编造 Combine 拒绝记录。
- “历史汇总回放”仅在已完成的真实结果上逐轮展示累计统计，不发起签名请求，不还原单条消息、逐 Attempt 事件或精确时间，不是实时监控。
- 正式实验默认内置从 `experiments/results/results.json` 提取的 2026-11-06 历史快照（100 条原始样本），并显示原实验时间、KeyID、环境、源文件 SHA-256。均值/P50/P95由原始样本重新计算，历史成功率明确标记为非实时。
- 不新增后台文件读取接口。替换磁盘上的实验结果不会自动更新已嵌入快照；可点击“加载实验 JSON”选择新的公开结果文件。文件仅在本地浏览器读取，不上传，限制 4 MiB、30–1000 个原始样本，检查编号、成功数、状态和耗时。无效文件不会替换上一份有效展示。
- 当前页面会话单独绘制真实签名延迟；少于 30 个成功样本时不显示 P50/P95。页面刷新清空会话，不混入正式历史测量。
- 安全用例展示请求、预期和实际 HTTP/验签结果；额外 GET 核对原交易、会话和签名未变。门限入参错误 400 与节点可用性故障 503 分开说明。

静态资源通过 `go:embed` 打包，必须重新构建并重启后端，单独刷新旧二进制提供的页面不会更新：

```powershell
# 在原服务窗口正常 Ctrl+C，保留原有启动环境配置，然后：
go build -o ./bin/backend.exe ./backend

# 使用原 C2.4 的环境配置启动，或使用下文的一键演示流程。
./bin/backend.exe
```

保留已有 8080 服务时，不要强制结束它；可在其他端口建立独立演示环境。浏览器验收截图及记录在 `experiments/ui-v2/`。

可选开发测试（运行页面不需要 Node.js）：

```powershell
node --check backend/web/app.js
node --test backend/web/app.test.cjs
```

前端测试会核对内置历史快照与当前 `experiments/results/results.json` 的来源摘要和全部样本。如果有意替换这份正式原始文件，应同步维护历史快照来源，或保留原始归档，不要把新结果冒充旧记录。

原生 HTML/CSS/JavaScript，由业务 Go 后端 `go:embed` 同源提供，不需要 Node.js、npm、CDN 或前端构建。已有 C2.4 启动流程不变；重新编译后端后打开它的根路径，例如 [本机监控台](http://127.0.0.1:8080)。后端入口仍是 `backend/`，不是另造 `cmd/backend/`。

## 一键实验和演示（PowerShell）

在 `threshold-mldsa-bench` 目录执行：

```powershell
go build -o ./bin/signer.exe ./cmd/signer
go build -o ./bin/launcher.exe ./cmd/launcher
go build -o ./bin/backend.exe ./backend
go build -o ./bin/experiments.exe ./cmd/experiments

# 自动选取节点端口、生成内存凭据、启动五节点与业务后端。
# 100 次正式签名后保留监控台 30 分钟；Ctrl+C 或到期清理本次进程。
./bin/experiments.exe -iterations=100 -port=8080 -preview=30m -output=experiments/results
```

如果 8080 已被自己的旧后端占用，先正常关闭旧后端，或改为 `-port=8081`。不要停止不认识的进程。如果系统 PATH 里没有 Go，用本机便携 Go 的完整路径代替 `go`：`& 'C:\Users\Lenovo\Desktop\金密杯\.tools\go\bin\go.exe' ...`。

命令打印 `Dashboard ready: http://127.0.0.1:...` 后打开该地址。先创建交易，保持 P0/P2/P4，点击“发起分布式签名”。也可以切换任意三个不同节点。测试使用模拟交易，无真实资金转移。

`experiments/results` 中的 `REPORT.md`、`results.json`、`samples.csv` 为公开实验输出。重新运行会替换同目录同名结果；保留多批实验请使用不同 `-output`。默认预热 5 次不计入统计，100 次正式样本均匀覆盖十种 3-of-5 组合。该驱动执行真实业务 HTTP 请求，额外使用公钥对每条签名及金额 +1 分副本分别验证；不调用旧集中式 SignMessage。

已有后端正常运行时不用启动新实验驱动；只需访问现有根页面。改动静态文件后需重新构建/重启 Go 后端，因为资源嵌在二进制中。

## 可观测口径

- `GET /api/cluster` 只公开模式、KeyID、3/5 参数、节点就绪状态、公开健康检查延迟及是否需要重新初始化。探测在后端完成，页面不会访问 `/internal/*`，不接收协调器 Token、初始化种子或私有状态。
- 签名接口兼容无请求体（默认 `[0,2,4]`），也接受 `{"participants":[0,2,4]}`。恰好三个不同编号，范围 0–4；在开始密码会话前校验。
- `POST /api/transactions/{id}/verify` 接收完整候选交易，与已保存签名独立验签，不修改原交易。安全面板分别检查金额篡改为 false、重复签名 409、两个参与方 400。后者是门限入参检查，不是节点宕机实验。
- 没有流式进度接口；“正在签名”只是请求等待态。Round1/2/3 是请求结束后真实返回的累计响应数，包括 Combine 拒绝后新尝试，不能称为实时轮次监控。
- `duration_ms` 是后端签名调用及独立验签耗时；页面性能图仅统计本页发起的请求，不是全局历史账本，不随刷新持久化。正式报告使用 100 次独立测量，nearest-rank P50/P95。
- `traffic.rounds` 按三轮分别记录 `http_sent_bytes`、`http_received_bytes`：协调器真实 `net.Conn.Write/Read` 字节，包含 HTTP/1.1 头、JSON、Base64 与分块编码；没有保存任何头部内容。另有 `protocol_response_bytes`，只计成功读取并绑定检查后的原始公开响应 Payload。全部尝试都累加。
- HTTP 统计排除健康/公钥/状态/Abort 请求、浏览器到后端的 API、TCP/IP 与链路层、TCP 重传。不把原始协议字节当成系统总网络流量。失败时字节数可能包含部分请求/响应；失败若没有完整轮次数则仍不展示虚构轮次。
- 本地基线模式不会伪造五个独立进程、网络轮次或网络流量。页面明确标注 Local baseline。

## 自动验收和故障证据

```powershell
$env:C24_SIGNER_BINARY = (Resolve-Path ./bin/signer.exe).Path
$env:C24_LAUNCHER_BINARY = (Resolve-Path ./bin/launcher.exe).Path
$env:C24_BACKEND_BINARY = (Resolve-Path ./bin/backend.exe).Path
$env:C23_SIGNER_BINARY = $env:C24_SIGNER_BINARY
$env:C23_LAUNCHER_BINARY = $env:C24_LAUNCHER_BINARY

go test -p 1 ./... -v -count=1 | Tee-Object -FilePath ./experiments/results/acceptance-test.log
go vet ./...
```

未设置二进制环境变量会跳过真实进程集成测试，不能以 SKIP 作为通过证据。真实进程测试覆盖并发重复提交、节点中断、执行后响应超时、确认 Abort 后新会话重试、未确认 Abort 时停止后续签名，以及 Combine 拒绝后全新的 Round1。测试只清理自己创建的进程。

交易、会话防重放和幂等保护仍限一次受控启动生命周期；节点重启后整体初始化新 KeyID，不恢复临时状态。当前没有认证用户、生产级秘密存储和跨重启业务账本。
