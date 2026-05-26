# 实施路线图（ROADMAP.md）

> 严格按周顺序执行。每周末停下来等 reviewer。

---

## 总览

| 周 | 主题 | 主要产出 | 高风险点 |
|----|------|---------|---------|
| W1-W2 | sub2api strip | `seller-relay/` 精简版 | wire/ent 重生成 |
| W3 | seller-relay 收尾 | SQLite + IPC | 数据库降级 |
| W4 | 协议库基础 | `protocol/` crate 公钥/签名 | canonical_json 两侧一致 |
| W5 | buyer-cli 骨架 | 本地 OpenAI 兼容 endpoint | 路由到 seller-relay |
| W6 | GitHub 同步层 | 多 fork pull/merge + bot | PR 签名验证 |
| W7 | commit-reveal | 端到端跑通假币竞价 | 各种 race condition |
| W8 | HTLC + state channel | Base 测试网清算 | 时序攻击 |
| W9 | buyer-tauri UI | fork cc-switch + 集成 | UX 完整度 |
| W10 | 联调 + alpha | 三人闭门测试 | 集成 bug |

**v0 不做**：carrier（洋葱路由）、Lightning、跨链、API 速率自动竞价、多语言 UI。

---

## W1 — sub2api 基线导入 + 路由 & handler 剪枝

### 目标

- fork sub2api 到 `seller-relay/` 目录
- 删除 admin / auth / payment / user 相关的路由 + handler
- 编译通过，已有 gateway e2e 测试通过

### 任务清单

按 `SUB2API_STRIP.md` 步骤 1-2 执行：

- [ ] 仓库初始化（`README.md` + `LICENSE` + `.gitignore`）
- [ ] git init 后第一个 commit
- [ ] clone sub2api 到 `seller-relay/`（按 SUB2API_STRIP.md §2）
- [ ] 验证基线编译通过
- [ ] Step 1：删除路由（一个 commit）
- [ ] Step 2：删除 admin/payment/user handler（一个 commit）

### 验收

```bash
cd seller-relay
rtk go build ./...   # 应通过（可能有 unused import warning）
rtk go vet ./...
git log --oneline seller-relay/ | wc -l  # 应至少 3 个 commit
```

提交 milestone 报告 `docs/progress/W1.md`，等 `APPROVED` 后进入 W2。

---

## W2 — service / repo / schema 剪枝 + wire 重跑

### 目标

- 按 SUB2API_STRIP.md Step 3-8 执行
- Ent schema 精简
- wire 和 ent 都能重新生成

### 任务清单

- [ ] Step 3：删除 service 层支付 / 订阅 / 推广
- [ ] Step 4：删除 service 层用户 / 认证 / admin
- [ ] Step 5：删除 service 层运营 / 仪表盘
- [ ] Step 6：删除 service 层渠道质量监测
- [ ] Step 7：重跑 wire generate
- [ ] Step 8：Schema 剪枝 + 重跑 ent generate

### 验收

```bash
cd seller-relay
rtk go build ./...
find . -name '*.go' ! -name '*_test.go' | wc -l   # 期望 < 350
rtk go test ./internal/pkg/...   # 全部通过
rtk go test ./internal/service/... -run "TestSticky|TestUMQ" # 通过
```

确认 `PITFALLS.md §11` 的 W1-W2 自检全部勾选。

---

## W3 — SQLite 降级 + IPC 接入

### 目标

- 去掉 PostgreSQL / Redis
- 加入 IPC（4 个 RPC + 1 个 notify）
- 启动后能与一个简单的 Rust 客户端通信

### 任务清单

- [ ] SUB2API_STRIP.md Step 9：SQLite 降级
- [ ] Migration 合并
- [ ] 新增 `internal/ipc/` 包
- [ ] 实现 `OpenEndpoint` RPC
- [ ] 实现 `CloseEndpoint` RPC
- [ ] 实现 `GetUsage` RPC
- [ ] 实现 `ListAccounts` RPC
- [ ] `cmd/server/main.go` 启动 IPC 监听
- [ ] 写一个 `tools/ipc-probe/` Rust 小工具（< 200 行）连 socket 调用所有 RPC

### 验收

```bash
# 启动 seller-relay
rtk go run ./seller-relay/cmd/server &

# 跑 IPC 测试工具
cargo run --bin ipc-probe -- --socket ~/.ai-bazaar/seller.sock

# 期望输出：所有 4 个 RPC 调用成功
# OpenEndpoint OK, endpoint_id=...
# GetUsage OK, tokens=0
# ListAccounts OK, count=0
# CloseEndpoint OK
```

```bash
# 数据库验证
sqlite3 ~/.ai-bazaar/seller.db ".tables"
# 应列出：accounts, api_keys, endpoints, usage_logs, settings, ...
# 不应有：users, payments, subscriptions, ...
```

```bash
# 依赖验证
rtk grep -r "redis" seller-relay/go.mod   # 应无（或仅在 unused 区）
rtk grep -r "lib/pq" seller-relay/go.mod   # 应无
```

---

## W4 — protocol crate 基础

### 目标

- 建立 Rust workspace
- 实现 `protocol/` crate 的身份、签名、canonical_json
- Rust 测试向量生成
- **新建** seller-relay 的 `internal/protocolcompat/` 包，作为 Go 端读 Rust 测试向量的回归点

### 任务清单

- [ ] `Cargo.toml` workspace 配置
- [ ] `protocol/` crate skeleton
- [ ] `identity.rs`：keypair 生成（SLIP-0010 + BIP-39）、序列化、fingerprint（BLAKE3-128）
- [ ] `signature.rs`：签名 / 验证 / canonical_json（**直接对 canonical bytes 签，不预 hash**）
- [ ] `messages.rs`：所有消息类型（Listing / Request / Commit / Reveal / Tx / Ticket）
- [ ] `commit_reveal.rs`：commitment 计算（canonical CBOR + BLAKE3 32 字节）
- [ ] 单元测试覆盖率 ≥ 85%
- [ ] proptest：签名/验证 roundtrip、commitment 单向性、reveal 加解密 roundtrip
- [ ] 测试向量生成 → 写入 `protocol/tests/vectors/`：
  - 5 条 listing 签名向量
  - 3 条 commitment 向量
  - 3 条 reveal 加密向量
  - 3 条 ticket 签名向量
- [ ] **新建** `seller-relay/internal/protocolcompat/`（Go 包）：
  - 加载 Rust 生成的 `tests/vectors/` 目录
  - 用 Go 重新计算签名 / commitment / canonical_json 并断言**字节级**等价
  - 一个测试入口：`go test ./internal/protocolcompat/`

### 验收

```bash
cargo test -p protocol
cargo clippy -p protocol --all-targets -- -D warnings
cargo tarpaulin -p protocol --out Stdout | grep -E "85\.|9[0-9]\.|100\." # 覆盖率达标

# Go 端读测试向量验证（W6 之前简单做一下）
cd seller-relay
go test ./internal/protocolcompat/ # 一个简单的兼容测试
```

### 风险

- canonical_json 在 Rust serde_json 和 Go encoding/json 上行为不同
- **必须**用 `tests/vectors/` 验证两侧字节一致

---

## W5 — buyer-cli 骨架 + 本地 OpenAI 兼容 endpoint

### 目标

- buyer-cli 启动后在 `127.0.0.1:11434` 暴露 OpenAI 兼容 endpoint
- 不实际竞价，用一个静态配置文件指定上游 seller-relay
- Cursor / Claude Code 能用这个 endpoint 完成一次请求

### 任务清单

- [ ] `buyer-cli/` Rust binary
- [ ] axum 路由：
  - `POST /v1/chat/completions` → 转发到 seller-relay
  - `POST /v1/messages` → 转发到 seller-relay
  - `GET /v1/models` → 静态返回
- [ ] 配置文件 `~/.ai-bazaar/buyer.toml`：
  ```toml
  [endpoints.test]
  url = "http://127.0.0.1:8000"
  api_key = "sk-test-static-key"
  ```
- [ ] 流式响应正确转发（SSE）
- [ ] 集成测试：buyer-cli + seller-relay 跑通一次 Claude API 请求

### 验收

```bash
# 启动 seller-relay（W3 完成的）
rtk go run ./seller-relay/cmd/server &

# 启动 buyer-cli
cargo run -p buyer-cli &

# 测试
curl http://127.0.0.1:11434/v1/chat/completions \
  -H "Authorization: Bearer dummy" \
  -d '{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}'

# 期望：流式响应正常返回
```

---

## W6 — GitHub 同步层 + Registry Bot

### 目标

- protocol crate 加 `github_sync` 模块
- 实现 manifest 解析、多 fork pull、视图合并
- 简易 registry bot（PR auto-merge）

### 任务清单

- [ ] `protocol/src/github_sync.rs`
- [ ] manifest.json 解析
- [ ] 多 fork shallow clone + 增量 fetch（用 git2 或 octocrab）
- [ ] 视图合并：按 quorum 选择
- [ ] 不一致检测 + 标记
- [ ] `tools/registry-bot/`（Rust）
- [ ] bot 监听 PR webhook → 校验签名 → auto-merge
- [ ] bot 单元测试 + 集成测试（用 fake GitHub API）

### 验收

```bash
# 起一个 fake registry（用 gitea / gitlab 容器，或 wiremock github API）
docker compose -f e2e/docker-compose.fake-github.yml up -d

# 用 manifest 指向 fake registry
echo '{"pv":"0.1.0","registry_forks":["http://localhost:3000/org/ai-bazaar-registry"],"min_quorum":1}' > /tmp/manifest.json

# 跑 sync 测试
cargo test -p protocol --test github_sync -- --nocapture

# 期望：能 pull、能 merge、能检测不一致
```

---

## W7 — commit-reveal 协议端到端

### 目标

- 完整跑通密封竞价流程（**Vickrey / second-price**）
- 模拟 3 个 seller 竞价同一个 request
- 最低价 winner，付款 = 第二低价

### 任务清单

- [ ] `protocol/src/commit_reveal.rs` 完整状态机
- [ ] `buyer-cli` 加 `request` 子命令：生成 request 并 PR 到 registry
- [ ] `seller-ctl` 加 watch 模式：监听 registry 新 request，自动出价
- [ ] `seller-ctl` 配置文件：定价策略（Vickrey 下的占优策略 = 报真实成本，配置项暴露 markup ratio）
- [ ] buyer-cli watch reveal：deadline 到达后**用 Vickrey 规则**算 winner / final_price
  - 实现 §3.5 三种边界：1 reveal / 多 reveal / 并列
- [ ] 落选者审计逻辑：seller-ctl 收到 tx 后验证自己 commitment 在 `all_commitments`，且 `final_price` ≤ 自己 reveal 价
- [ ] e2e 测试：fake registry + 3 个 seller + 1 个 buyer
- [ ] **失败路径**测试：
  - 卖家 commit 后不 reveal → forfeit
  - 卖家 reveal 价格与 commit 不匹配 → 拒绝 + 声誉惩罚
  - 卖家报价 > max_price → 自动失格
  - **新增**：仅 1 个有效 reveal → final_price 等于 max_price（不是 reveal 价）
  - **新增**：2 个 seller 报同样最低价 → BLAKE3 决定性 winner

### 验收

```bash
cargo test -p protocol --test commit_reveal_e2e -- --nocapture

# 期望日志（Vickrey 例子）：
# [buyer] published request req-001 (max 5.00 USDC, 100k tokens)
# [seller-a] committed (price=4.50)
# [seller-b] committed (price=4.20)
# [seller-c] committed (price=4.80)
# [seller-a] revealed
# [seller-b] revealed
# [seller-c] revealed
# [buyer] winner: seller-b (revealed 4.20 USDC)
# [buyer] Vickrey final price: 4.50 USDC (= second-lowest reveal)
# [tx] created tx-xxx with winning_bid=4200000 final_price=4500000
# [seller-c] audit: my reveal 4.80 ≥ final 4.50 → OK
# [seller-a] audit: my reveal 4.50 ≥ final 4.50 → OK (boundary)
```

---

## W8 — HTLC + state channel

### 目标

- 部署 HTLC 合约到 Base 测试网
- buyer-cli 创建 HTLC、seller-ctl 监控、claim
- state channel ticket 累积 + 终结上链

### 任务清单

- [ ] `chain/contracts/` Solidity HTLC 合约（Foundry / Hardhat）
- [ ] OpenZeppelin ReentrancyGuard 集成
- [ ] 单元测试：deposit / claim / refund / 重放攻击
- [ ] 部署脚本（Base Sepolia 测试网）
- [ ] `chain/client/` Rust 调用代码（alloy）
- [ ] state channel ticket 签名 / 验证 / 累积
- [ ] buyer-cli 集成：tx 后自动 HTLC.deposit
- [ ] seller-ctl 集成：监听 HTLC.deposit、开始服务、累积 ticket
- [ ] 卖家 claim 流程
- [ ] 买家 refund 流程
- [ ] "gas dust" 转账

### 验收

```bash
# 部署合约到 Base Sepolia
cd chain && forge script script/Deploy.s.sol --rpc-url $BASE_SEPOLIA_RPC --broadcast

# 端到端测试
cargo test -p chain-client --test e2e_settlement -- --nocapture

# 期望：
# - 买家 deposit 0.5 USDC + 0.0001 ETH gas
# - 卖家服务 100k tokens
# - ticket seq=10, cumulative_tokens=100000, cumulative=0.45 USDC
# - 卖家 claim：链上余额变化 +0.45 USDC, 退还买家 0.05 USDC
```

---

## W9 — buyer-tauri UI

### 目标

- fork cc-switch
- 改造 UI：显示市场列表、发起 request、跟踪 HTLC 状态
- 与 buyer-cli 通过 IPC 通信

### 任务清单

- [ ] fork cc-switch 到 `buyer-client/`
- [ ] 删掉 cc-switch 的广告位 / 赞助商
- [ ] UI 新增页面：
  - **Marketplace**：sellers 列表（按服务过滤）
  - **Active Requests**：进行中的 request + 倒计时
  - **Active Endpoints**：可用的本地 endpoint URL + 剩余配额
  - **Wallet**：USDC 余额、HTLC 状态、历史 tx
  - **Reputation**：自己的评价、收到的评价
- [ ] buyer-cli 新增 RPC：`ListMarketplace` / `SubmitRequest` / `GetActiveEndpoints`
- [ ] Tauri ↔ buyer-cli IPC
- [ ] keychain 集成（密钥存储）

### 验收

手工跑一遍完整 UI 流程：

1. 启动 buyer-cli + buyer-client
2. 创建钱包 / 输入助记词
3. 浏览 marketplace
4. 发起 request
5. 看到 bids 实时进入
6. reveal 后选 winner
7. HTLC deposit
8. 看到 endpoint 出现
9. 在 Cursor 配置该 endpoint 实际用
10. 服务结束 claim / refund

### 验收（自动化部分）

```bash
cargo tauri build  # 三平台都能构建
cargo test -p buyer-client  # 单元测试通过
```

---

## W10 — 联调 + alpha 邀请

### 目标

- 三人闭门测试（reviewer + 你 + 一个独立 alpha 用户）
- 修关键 bug
- 写 alpha 文档

### 任务清单

- [ ] 把所有组件打包成发布物（macOS / Linux / Windows）
- [ ] 写 `INSTALL.md`（如何 seller 上线、如何 buyer 试用）
- [ ] 写 `KNOWN_ISSUES.md`
- [ ] 准备一个 demo manifest（reviewer 维护）
- [ ] 跑 3 轮 e2e 真实交易（每轮上限 $1）
- [ ] 记录所有 bug
- [ ] 修 P0 bug
- [ ] 发布 v0.1.0-alpha tag

### 验收

reviewer 与 alpha 用户分别能：

- 安装客户端
- 完成密钥生成
- seller 上线挂牌
- buyer 发起 request 并成交
- 用到的 endpoint 在 Cursor / Claude Code 里正常工作
- HTLC 正确清算

---

## 路线图之外（v1+ 预留）

- v1.0：carrier 洋葱路由、声誉系统正式版、PR auto-merge 性能优化
- v1.1：跨协议路由（开 apicompat）
- v1.2：移动端（iOS / Android）
- v2.0：Lightning Network 接入

不要在 v0 提前实现这些。

---

## 进度跟踪

每周末以 PR 形式提交 `docs/progress/W{N}.md`（受 ruleset 保护，必须走 PR
+ reviewer approval，与代码 PR 一致）。模板见 `docs/progress/TEMPLATE.md`。
