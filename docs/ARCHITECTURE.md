# 架构决策记录（ADR）

> 本文档记录所有架构层面的"为什么这么选"。改任何决策前必须 PR 修改本文档。

---

## ADR-001：Polyglot（Rust + Go）

### 决策

| 组件 | 语言 | 理由 |
|------|------|------|
| `seller-relay` | **Go** | fork sub2api，保留 utls 反指纹 + 协议互转 + 沉淀的反封号经验 |
| `seller-ctl` | **Rust** | 控制 seller-relay 的"大脑"，做签名 / 竞价 / GitHub 同步 |
| `buyer-client` | **Rust + Tauri** | fork cc-switch，桌面 UI |
| `buyer-cli` | **Rust** | 无头版本，给服务器场景用 |
| `protocol` | **Rust crate** | 协议参考实现 |
| `chain/` Solidity + Rust 客户端 | **Solidity + Rust** | HTLC 合约 + alloy/ethers-rs 调用 |

### 备选方案与否决理由

**全 Rust**：放弃。理由：
- sub2api 的 utls 反指纹生态 Rust 暂时追不上（rustls 不支持任意 JA3 复刻）
- 重写 ~13500 行 Go 的反爬虫 / sticky session / OAuth 漂移逻辑 = 8–12 周 + bug
- 失去 `git pull upstream` 跟进 sub2api 的能力

**全 Go**：放弃。理由：
- Tauri 是桌面 UI 最优解，Go 的 Fyne/Wails 体验明显差一档
- alloy / ethers-rs 比 go-ethereum 在 HTLC + state channel 场景下显著更现代
- Rust 的类型系统对协议状态机更友好（commit-reveal 错误状态 compile-time 拦截）

### 跨语言通信

**坚决不用 FFI（cgo / Rust bindgen）**。两侧通过 IPC + JSON 通信：

```
seller-ctl (Rust)  ──── Unix Socket (JSON-RPC) ────  seller-relay (Go)
buyer-cli  (Rust)  ──── Unix Socket (JSON-RPC) ────  buyer-tauri  (Rust)
```

IPC 协议在 `PROTOCOL.md §6 Control Plane` 定义。Go 这边只暴露 4 个 RPC：

- `OpenEndpoint(buyer_pubkey, model, quota)` → `endpoint_url, api_key`
- `CloseEndpoint(endpoint_id)`
- `GetUsage(endpoint_id)` → `{tokens_used, requests}`
- `ListAccounts()` → `[{platform, status, capacity_remaining}]`

> **设计理由**：Go 这边的 sub2api fork 不应该知道"市场"概念，它只是一个能动态开关 endpoint 的本地代理。所有"商业逻辑"在 Rust 这边。

---

## ADR-002：GitHub 作为去中心化发现层

### 决策

用 **多个 fork 镜像 + 签名 PR** 的方式做发现层，而不是引入 IPFS / libp2p / blockchain：

- 一个"参考仓库" `ai-bazaar/registry`（任何中立组织都可以建）
- 每个 fork 都是平等镜像，客户端读取多个 fork 合并视图
- 所有写入是 PR，bot 自动 merge 通过签名校验的 PR
- bot 挂了也能用：客户端可以直接读 PR 列表

### 备选方案与否决理由

**IPFS / libp2p**：放弃。理由：
- 用户群（开发者）有 GitHub 账号但不一定愿意装 IPFS
- pinning 服务的可用性差，发现延迟高
- v1 可以加 IPFS 作为 manifest 备份，但不作为主链路

**自建去中心化链**：过度工程。

**Tor hidden service**：放弃。理由：
- 卖家 / 买家挂载点变化频繁，发现仍然需要中心化目录
- 可以作为 endpoint 通信层选项，但不是发现层

### 抗审查策略

- `manifest.json` 列出 N 个镜像 fork（≥3，建议 ≥7）
- IPFS / Arweave 上传 manifest 的备份（不依赖，仅 fallback）
- 用户可以在 buyer-client 里**手动添加 fork URL**
- 主仓库被 DMCA → 任何 fork 都能继续工作

---

## ADR-003：资金结算用 Base + USDC + HTLC

### 决策

- **链**：Base（Coinbase L2，gas 极低）
- **资产**：USDC（受监管的稳定币，比 USDT 法务风险低）
- **托管机制**：HTLC（Hash Time-Locked Contract）+ state channel
- **结算频率**：链下签状态、链上仅开 / 关 / 仲裁

### 流程

```
1. 买家在链上 HTLC.deposit(seller_pubkey, amount, hashlock, timelock=1h)
2. 卖家开始服务，每提供 1k tokens：买家签发 ticket{tokens, amount, sig}
3. ticket 累计在卖家本地（不上链）
4. 服务结束 OR timelock 到，卖家拿最后一张 ticket + preimage 调 HTLC.claim()
5. 余额自动退回买家
6. 服务中断：买家在 timelock 到期后调 HTLC.refund() 取回全款
```

### 备选与否决

- **Lightning Network**：v2 考虑。理由：Lightning 的 channel 管理对终端用户太复杂。
- **以太坊主网**：放弃。gas 太贵。
- **Solana**：放弃。Rust 生态对 Solana 的 HTLC 实现不如 EVM 系成熟。

---

## ADR-004：身份系统 = Ed25519 公钥

### 决策

- **不用** JWT / OAuth / 账号密码 / 邮箱
- **唯一身份** = Ed25519 公钥
- GitHub 账号只用来"领认公钥"（在个人主页 README 放公钥指纹），可选不强求
- 所有协议消息必须 detached signature

### 工程要求

- 私钥默认存系统 keychain（macOS Keychain / Windows Credential Store / Linux Secret Service）
- 备份导出格式：BIP39 助记词（24 词）派生 → Ed25519 keypair
- 派生路径：`m/44'/0'/0'/0/0`（自定义 coin type，避免与比特币撞）
- 签名算法：纯 Ed25519 + SHA-512（**不要用** Ed25519ph / Ed25519ctx）

---

## ADR-005：seller-relay 数据库降级到 SQLite

### 决策

- 不用 PostgreSQL（sub2api 默认）
- 不用 Redis（sub2api 默认）
- 单文件 SQLite（modernc.org/sqlite，纯 Go，无 CGO）
- WAL mode 开启，自动 vacuum

### 理由

- 卖家是个人用户，PostgreSQL 部署成本太高
- Redis 仅用于 cache，SQLite + 内存 map 就够
- 单文件方便备份 / 跨机器迁移

### 例外

- 如果 sub2api fork 的某个 service 强依赖 PostgreSQL（如某些 JSONB 查询），**降级该 service 到内存实现**，不要把 PostgreSQL 拉回来
- `SUB2API_STRIP.md` 列出了所有需要降级的点

---

## ADR-006：Rust 依赖白名单（W1-W10）

只允许以下 crate。要加新 crate 必须 PR 改本文档。

### 协议 / 密码学

```toml
ed25519-dalek = "2"           # Ed25519 签名
sha2 = "0.10"                 # SHA-256 / SHA-512
blake3 = "1"                  # 快速哈希（commitments）
snow = "0.9"                  # Noise protocol framework
age = "0.10"                  # 文件加密
rand = "0.8"                  # CSPRNG
zeroize = "1"                 # 内存擦除
chacha20poly1305 = "0.10"     # AEAD
hkdf = "0.12"                 # HKDF
bip39 = "2"                   # 助记词
hkd32 = "0.7"                 # 层次化派生
```

### 序列化

```toml
serde = { version = "1", features = ["derive"] }
serde_json = "1"
serde_with = "3"              # base64 / hex 字段
ciborium = "0.2"              # CBOR（紧凑二进制，用于 commit）
```

### 异步运行时

```toml
tokio = { version = "1", features = ["full"] }
tokio-util = "0.7"
futures = "0.3"
async-trait = "0.1"
```

### HTTP / 网络

```toml
reqwest = { version = "0.12", features = ["json", "stream", "rustls-tls"] }
hyper = "1"
axum = "0.7"                  # buyer-cli 本地 OpenAI 兼容 endpoint
tower = "0.5"
tower-http = "0.6"
```

### 持久化

```toml
rusqlite = { version = "0.32", features = ["bundled"] }
sled = "0.34"                 # 仅 protocol crate 用于本地状态缓存
keyring = "3"                 # 跨平台 keychain
```

### GitHub

```toml
octocrab = "0.42"             # GitHub REST API
git2 = "0.19"                 # libgit2 binding
```

### 区块链

```toml
alloy = { version = "0.7", features = ["full"] }   # ethers-rs 继任者
```

### CLI / 日志

```toml
clap = { version = "4", features = ["derive"] }
tracing = "0.1"
tracing-subscriber = "0.3"
color-eyre = "0.6"
```

### 测试

```toml
[dev-dependencies]
proptest = "1"
insta = "1"
mockall = "0.13"
tempfile = "3"
wiremock = "0.6"
```

### 禁用清单

- ❌ `openssl`（用 rustls 系列）
- ❌ `secp256k1` 裸用（要用就用 alloy 封装好的）
- ❌ `tokio-tungstenite` 直接用（用 reqwest websocket feature）
- ❌ `actix-web`（统一用 axum）
- ❌ `diesel`（用 rusqlite 直接 SQL）
- ❌ 任何 `*-sys` 强依赖 C 库的，除非已在白名单

---

## ADR-007：Go 依赖策略

- **不主动加 Go 依赖**，只用 sub2api fork 自带的
- 如果删除某些 sub2api 模块导致依赖变孤儿，跟着删
- 唯一可能新增：`golang.org/x/sys/unix` 的 socket helper（IPC 用）

---

## ADR-008：测试策略

### 单元测试

- 所有 `protocol/` crate 的函数必须有单元测试
- 密码学函数额外加 property-based test（proptest）
- 反例：恶意输入、过期签名、replay 攻击

### 端到端测试

`e2e/` 目录下用 docker-compose 拉起：
- 一个 seller-relay + seller-ctl
- 两个 buyer-cli（模拟竞争）
- 一个 git server（fake GitHub）
- 一个 anvil（Base 测试网模拟）

跑完整业务流并断言：
- 价格最低的 bid 中标
- 中标价 ≤ 其他 bid 价
- token 计量误差 < 1%
- HTLC 在正常 / 异常路径下都能正确清算

### 覆盖率

- `protocol/` 至少 85%
- `seller-ctl` / `buyer-cli` 至少 70%
- `seller-relay` fork 部分不强求（继承 sub2api 测试）

---

## ADR-009：版本号策略

- 协议版本：`PROTOCOL.md` 顶部 `protocol_version: 0.x.y`，所有消息必须带 `pv` 字段
- 客户端版本：semver，主版本号 == 协议主版本号
- 不兼容协议变更必须升 protocol_version，并在 6 周内允许双版本共存

---

## ADR-010：日志与隐私

- 日志**绝不**记录：API key、prompt 内容、用户 IP（除非 debug 模式 + 明示同意）
- 日志可以记录：时间戳、token 数、上游平台、错误码、链上 tx hash
- 默认 log level：`info`；`debug` 必须显式开启
- 所有 `tracing::debug!` 中包含敏感字段的必须用 `?` 或 `redact()` 包装

---

## 待定（OPEN）

- ❓ buyer-client 的前端框架（React vs Vue vs Svelte）— W5 决定
- ❓ 是否引入 carrier（洋葱路由）— v1 决定，不在 W1-W10
- ❓ Lightning 集成 — v2 决定
- ❓ 跨链支持 — v2 决定
