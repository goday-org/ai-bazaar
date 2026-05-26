# Review 检查点（REVIEW.md）

> Reviewer 会在每个 milestone 严格按本文档检查。Codex 应该自查后再提交。

---

## Severity 定义

Reviewer 在 review 中标的 issue 用三档 severity，决定是否阻塞当前 milestone：

| 级别 | 含义 | 处理 |
|------|------|------|
| **P0** | 阻塞性问题：触碰 PITFALLS 🔴 / 协议契约违反 / 资金安全 / secret 泄露 / 编译失败 / 测试不通过 | **当前 PR 修复**，修完才能合 |
| **P1** | 严重但不阻塞：UX bug、API 设计欠佳、覆盖率明显不足、文档不符 | 记入 `KNOWN_ISSUES.md`，**下个 milestone 内修** |
| **P2** | 风格 / 优化建议：命名、注释清晰度、性能微调 | 可选修；不写 KNOWN_ISSUES |

Reviewer 在 inline comment 里用 `[P0]` / `[P1]` / `[P2]` 前缀标注。Codex 必须按级别处理；不同意分级 → 在 PR 评论里讨论 **再** 修，不要默默忽略。

---

## 通用 Review 标准（每个 milestone 都过一遍）

### Commit 质量

- [ ] Commit message 用 conventional 风格：`type(scope): description`
- [ ] 一个 PR < 1500 行 diff（W1-W2 例外允许 5000 行）
- [ ] 没有 `WIP` / `tmp` / `fixup` / `xxx` / 中文标点 的 commit
- [ ] 任何 `DEVIATION` 都在 commit body 里说明

### 代码质量

- [ ] Rust：`cargo clippy --all-targets -- -D warnings` 通过
- [ ] Rust：`cargo fmt --check` 通过
- [ ] Go：`go vet ./...` 通过
- [ ] Go：`gofmt -l .` 输出为空
- [ ] 没有 `TODO`/`FIXME`/`XXX` 没标注 issue 链接

### 测试

- [ ] 新增功能必带测试
- [ ] CI 全绿（GitHub Actions / 本地等价命令）
- [ ] 测试覆盖率达标（见 ARCHITECTURE.md ADR-008）

### 安全

- [ ] `git grep -E '(sk-|GOCSPX-|0x[a-fA-F0-9]{64}|api_key=)'` 仅在文档 / 注释中出现
- [ ] 没有 `.env` / `secrets.json` 等被 commit
- [ ] 没有 `unsafe` 块（除非有书面同意）
- [ ] 私钥相关结构都 Zeroize
- [ ] 日志不含敏感信息

### 文档

- [ ] README 更新（如果有新二进制 / 新命令）
- [ ] `docs/progress/W{N}.md` 提交
- [ ] 任何架构层面修改在 `ARCHITECTURE.md` 里有对应 ADR 更新

---

## W1 检查点

**主题**：sub2api 基线导入 + 路由 & handler 剪枝

### 必查

- [ ] `seller-relay/LICENSE` 存在（LGPL-3.0，来自上游）
- [ ] `seller-relay/README.md` 顶部标明 "Forked from Wei-Shaw/sub2api"
- [ ] `seller-relay/go.mod` 没引入新依赖
- [ ] `seller-relay/internal/server/routes/` 下不应有 `admin.go` / `auth.go` / `user.go` / `payment.go`
- [ ] `seller-relay/internal/handler/admin/` 目录不存在
- [ ] `seller-relay/internal/handler/payment_*.go` 不存在
- [ ] `git log seller-relay/` 显示按 SUB2API_STRIP.md Step 顺序提交
- [ ] `rtk go build ./...` 在 seller-relay 目录通过
- [ ] `rtk go vet ./...` 通过

### 抽查（reviewer 会随机看）

- [ ] 抽查一个保留的 service 文件，看是否未被无意修改
- [ ] 抽查 wire.go，看是否仍有支付 / 用户 provider 残留

### 阻塞放行

- 没有删错任何 `internal/pkg/claude/` `internal/pkg/openai/` `internal/pkg/antigravity/` `internal/pkg/tlsfingerprint/` 下的文件

---

## W2 检查点

**主题**：service / repo / schema 剪枝 + wire/ent 重跑

### 必查

- [ ] `seller-relay/internal/service/payment_*.go` 不存在
- [ ] `seller-relay/internal/service/subscription_*.go` 不存在
- [ ] `seller-relay/internal/service/auth_*.go` 不存在
- [ ] `seller-relay/internal/service/admin_*.go` 不存在
- [ ] `seller-relay/internal/service/ops_*.go` 不存在
- [ ] `seller-relay/internal/service/affiliate_*.go` 不存在
- [ ] `seller-relay/ent/schema/payment_*.go` 不存在
- [ ] `seller-relay/ent/schema/subscription_plan.go` 不存在
- [ ] `wire_gen.go` 是 fresh 生成的（时间戳合理）
- [ ] `ent/` 下所有生成代码与 schema 一致
- [ ] `rtk go test ./internal/pkg/...` 全部通过
- [ ] `rtk go test ./internal/service/... -run "TestSticky|TestUMQ|TestAccount|TestRate"` 通过

### 量化指标

- [ ] `find seller-relay -name '*.go' ! -name '*_test.go' | wc -l` ≤ 350
- [ ] 总代码行数 ≤ 60,000

### 阻塞放行

- PITFALLS.md §11 W1-W2 自检清单全部勾选
- 必须保留的硬常量未被修改：

```bash
# 验证 Claude 常量未改
git diff <baseline> -- seller-relay/internal/pkg/claude/constants.go
# 期望：无 diff 或仅 import path 改动

# 验证 OpenAI 常量未改
git diff <baseline> -- seller-relay/internal/pkg/openai/oauth.go
# 期望：无 diff 或仅 import path 改动

# 验证 Antigravity 常量未改
git diff <baseline> -- seller-relay/internal/pkg/antigravity/oauth.go
# 期望：无 diff 或仅 import path 改动
```

---

## W3 检查点

**主题**：SQLite 降级 + IPC 接入

### 必查

- [ ] `seller-relay/go.mod` 不含 `github.com/lib/pq` / `github.com/jackc/pgx`
- [ ] `seller-relay/go.mod` 不含 `github.com/redis/go-redis`（或 `Enabled=false` 默认）
- [ ] `seller-relay/go.mod` 含 `modernc.org/sqlite`
- [ ] 启动后 `~/.ai-bazaar/seller.db` 创建成功
- [ ] `sqlite3 ~/.ai-bazaar/seller.db ".tables"` 不含 `users` / `payments` / `subscriptions`
- [ ] IPC socket `~/.ai-bazaar/seller.sock` 权限 0600
- [ ] 4 个 RPC 全部可用
- [ ] `tools/ipc-probe` 能跑通

### 阻塞放行

- 没有把已经删的功能加回来"过 wire"
- sticky session / UMQ / 错误计数器逻辑保留
- IPC server 在异常输入下不 panic

### 抽查

- 关掉 seller-relay 进程，重启后能从 SQLite 恢复 accounts / api_keys
- 同时发 50 个并发 RPC 不会卡死

---

## W4 检查点

**主题**：protocol crate 基础

### 必查

- [ ] `Cargo.toml` workspace 配置正确，所有 crate 列出
- [ ] `protocol/Cargo.toml` 依赖严格遵守 ADR-006 白名单
- [ ] `cargo clippy -p protocol --all-targets -- -D warnings` 通过
- [ ] `cargo test -p protocol` 全部通过
- [ ] `protocol/tests/vectors/` 存在并提交
- [ ] 测试向量包含：5 个签名向量、3 个 commitment 向量、3 个 reveal 加密向量
- [ ] proptest 至少覆盖：sign/verify roundtrip、commitment 单向性

### 跨语言一致性

```bash
# Reviewer 会跑这套
cd seller-relay
rtk go test ./internal/protocolcompat/ -v

# 期望：能验证 Rust 生成的所有测试向量
```

### 阻塞放行

- canonical_json 在 Rust 和 Go 输出 byte-for-byte 一致
- 私钥结构 Zeroize、不出现在 Debug

### 抽查

- 随机选 3 个消息类型，看 schema 与 PROTOCOL.md §3 是否完全一致
- 字段命名、可选性、长度限制都对得上

---

## W5 检查点

**主题**：buyer-cli 骨架 + 本地 OpenAI 兼容 endpoint

### 必查

- [ ] `buyer-cli` 启动后监听 `127.0.0.1:11434`
- [ ] `/v1/chat/completions` 流式响应正确转发（chunk 不被 buffer）
- [ ] `/v1/messages` 流式响应正确转发
- [ ] 配置文件路径 `~/.ai-bazaar/buyer.toml`
- [ ] 错误透传：上游 401 / 429 等正确传给客户端
- [ ] 不在日志记录 prompt content

### 手工验证

- [ ] 用 `curl` 测试一次非流式请求
- [ ] 用 `curl --no-buffer` 测试一次流式请求（SSE）
- [ ] 把 Cursor base URL 改为 `http://127.0.0.1:11434/v1`，能正常工作

### 阻塞放行

- 流式响应延迟 < 100ms（每个 chunk）
- 日志清洗

---

## W6 检查点

**主题**：GitHub 同步层 + Registry Bot

### 必查

- [ ] `protocol/src/github_sync.rs` 实现
- [ ] manifest.json 解析
- [ ] 多 fork pull 并行执行（用 tokio::join!）
- [ ] 视图合并按 min_quorum 工作
- [ ] 不一致检测：单元测试模拟两个 fork 数据不同
- [ ] registry bot 校验签名 + schema 后才 merge
- [ ] bot 拒绝非法 PR（签名错 / schema 错 / 时间戳过期）

### e2e 测试

```bash
cd e2e
docker compose -f fake-github.yml up -d
cargo test -p protocol --test github_sync -- --nocapture
```

应该看到：

- 一个 PR 通过 → merged
- 一个 PR 签名错 → closed with reason
- 一个 PR 路径不对（如 sellers/wrong-fp.json） → closed
- 一个 PR 重复（同一文件 hash） → 拒绝

### 阻塞放行

- bot 不会 merge 任何未签名 / 签名错的 PR
- 不一致冲突时 UI 不崩溃

---

## W7 检查点

**主题**：commit-reveal 端到端

### 必查

- [ ] 状态机所有转换有单元测试
- [ ] commit-reveal 失败路径全部测试：
  - 重复 commit → 拒绝
  - 价格 > max_price → 自动失格
  - commit 后不 reveal → forfeit + 声誉惩罚
  - reveal 价格与 commit 哈希不一致 → 拒绝 + 声誉惩罚
- [ ] 3 seller 竞价场景：最低价中标
- [ ] all_commitments 在 tx 中完整列出

### e2e 验证

```bash
cargo test -p protocol --test commit_reveal_e2e -- --nocapture
```

输出必须显示：

- 3 个 seller 都 commit
- 3 个 reveal 中 1 个故意作弊（哈希不匹配）→ 被踢
- 最低价 seller 中标
- tx 双签名完整
- 所有 commitments 在 tx 中可见

### 抽查

- 中标 seller 的 `winning_bid_micro_usdc` 与其 reveal 价格一致
- `final_price_micro_usdc` 等于第二低 reveal 价（≥ 2 reveal 时）；等于 `max_price` 仅当只有 1 个有效 reveal
- 落选 seller 能"事后验证"自己的 reveal 价 ≥ `final_price_micro_usdc`

### 阻塞放行

- 竞价 race condition：两个 seller 同时 commit 同一 req_id，应一个成功一个失败
- 协议消息超时不接收
- 时间戳偏差超过 §2.4 容差（+5min / -24h）拒绝
- **Vickrey 边界正确**：单一 reveal → 付 `max_price`；并列最低 → 用 `BLAKE3(req_id ‖ sorted_seller_fps)` 确定性选 winner

---

## W8 检查点

**主题**：HTLC + state channel

### 必查（合约）

- [ ] OpenZeppelin ReentrancyGuard 集成
- [ ] `deposit` / `claim` / `refund` 三个入口
- [ ] timelock 检查正确
- [ ] hashlock 校验用 keccak256（EVM 原生）
- [ ] 单元测试：
  - 正常 claim（preimage 正确）
  - timelock 到期前 refund 拒绝
  - timelock 到期后 refund 成功
  - 重放攻击：同一 HTLC 不能被 claim 两次
  - 错误 preimage 拒绝
- [ ] foundry 测试覆盖率 ≥ 90%

### 必查（客户端）

- [ ] alloy 集成
- [ ] buyer 调 deposit + gas dust
- [ ] seller 监控 deposit 事件
- [ ] ticket signature 校验
- [ ] ticket seq 单调递增校验
- [ ] seller claim 流程
- [ ] buyer refund 流程

### 阻塞放行（关键安全）

- preimage 不复用
- buyer refund 不依赖 seller
- USDC transfer 失败时回滚（用 SafeERC20）
- 不允许 amount=0 的 deposit

### 测试网验证

- 部署到 Base Sepolia
- 完成至少 1 笔成功 claim、1 笔成功 refund
- 在 Etherscan 看到交易历史

---

## W9 检查点

**主题**：buyer-tauri UI

### 必查

- [ ] fork cc-switch 标注清楚
- [ ] 删除原 cc-switch 的赞助商页面
- [ ] 至少 5 个页面实现（Marketplace / Active Requests / Active Endpoints / Wallet / Reputation）
- [ ] 密钥从 keychain 读取
- [ ] keychain 写入有权限提示
- [ ] Tauri ↔ buyer-cli IPC 正常
- [ ] 三平台都能构建：macOS / Linux / Windows

### 手工验证（完整流程）

reviewer 跑一遍：

1. 启动 buyer-cli + buyer-client
2. 首次启动 → 引导生成钱包 → 助记词显示
3. 重启 → 助记词导入
4. 浏览 marketplace（fake registry）
5. 发起 request
6. 看 bids 涌入
7. winner 选定
8. HTLC deposit（点按钮）
9. 看到 endpoint 出现在 Active Endpoints 页
10. 复制 endpoint URL 到 Cursor 配置
11. 用 Cursor 跑一次代码补全
12. 看到 token 计量在 UI 更新
13. 关闭 endpoint → 看到 HTLC 自动 claim

### 阻塞放行

- UI 不崩溃（崩了 reviewer 直接打回）
- 助记词不进剪贴板
- 助记词显示页面有截图警告

---

## W10 检查点

**主题**：联调 + alpha 发布

### 必查

- [ ] 三平台发布 binaries（macOS x64, macOS arm64, Linux x64, Windows x64）
- [ ] `INSTALL.md` 完整
- [ ] `KNOWN_ISSUES.md` 提交
- [ ] 至少 3 笔真实测试网交易完成
- [ ] alpha 用户独立完成一次端到端流程（无 reviewer 干预）

### 阻塞放行

- 没有 P0 bug
- 安全自检全部通过：

```bash
# 1. 没有 secret 泄露
git secrets --scan-history

# 2. 没有 unsafe
grep -r "unsafe" --include='*.rs' protocol/ buyer-cli/ seller-ctl/ buyer-client/
grep -r "unsafe.Pointer" --include='*.go' seller-relay/

# 3. 依赖审计
cargo audit
gosec ./...

# 4. SBOM
cargo cyclonedx > sbom-rust.json
go-licenses report ./... > licenses-go.txt
```

### 发布

- 打 tag `v0.1.0-alpha`
- GitHub Release 包含三平台 binary + SHA256
- 发布 changelog

---

## 出现问题时的处理

### 如果 reviewer 标 P0 问题

- Codex 必须立即停止下一周工作
- 在当前 PR 修复
- 修复后再次提交 review 申请

### 如果 reviewer 标 P1 问题

- 记录到 `KNOWN_ISSUES.md`
- **下个** milestone 之内修完
- 不阻塞当前 milestone 放行

### 如果 reviewer 标 P2 问题

- 可选修
- 不进 KNOWN_ISSUES，不阻塞放行

### 如果 Codex 不同意 reviewer 分级

- 在 PR 评论里说明理由
- 引用具体代码 / 测试 / 文档
- reviewer 二次回复后服从决定

---

## Review 节奏

Reviewer 承诺：

- Codex 提交 milestone 报告后 **24 小时内**给出反馈
- 反馈分为：
  - `APPROVED 进入 W{N+1}`
  - `CONDITIONAL：修以下 X 点后再进入下一周`
  - `BLOCKED：需要重新设计某部分，详见 ...`

Codex 不要催 review，但 24h 未回复可以发提醒。

---

## 最终验收（v0.1.0-alpha 发布前）

reviewer 会跑完整套清单：

- [ ] 所有 10 个 milestone 都 APPROVED
- [ ] PITFALLS.md 所有 🔴 项无违反
- [ ] 协议测试向量在 Rust / Go 两侧字节一致
- [ ] HTLC 合约通过外部审计基本检查（reviewer 用 mythril / slither 跑）
- [ ] 一个独立 alpha 用户完成端到端流程
- [ ] 三平台 binary 校验通过
- [ ] 文档完整（README / INSTALL / PROTOCOL / KNOWN_ISSUES）

通过即发布 v0.1.0-alpha。
