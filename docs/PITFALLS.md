# 致命陷阱清单（PITFALLS.md）

> 这些是踩了就会立刻挂掉的坑。每一条都标了严重级：
>
> - 🔴 **致命**：踩了就被上游封号 / 资金被盗 / 系统失能
> - 🟡 **严重**：会导致功能错乱、用户体验崩坏
> - 🟢 **小心**：值得注意但不致命
>
> Codex 完成任何 milestone 前必须自检对应级别的所有项。

---

## 1. 上游平台反爬虫（🔴）

### 1.1 Claude / Anthropic 必须 1:1 复刻 CLI

来源：sub2api `internal/pkg/claude/constants.go`

| 字段 | 当前值 | 备注 |
|------|--------|------|
| User-Agent | `claude-cli/2.1.92 (external, cli)` | **跟 CLI 版本同步漂移** |
| `anthropic-beta`（OAuth, 非 Haiku） | 见下方完整列表 | **顺序敏感** |
| `anthropic-beta`（Haiku） | 仅 `oauth-2025-04-20,interleaved-thinking-2025-05-14` | **不能带 claude-code-** |
| `anthropic-beta`（API-key） | **不带** `oauth-2025-04-20` | 否则 400 |
| `X-Stainless-Lang` | `js` | |
| `X-Stainless-Package-Version` | `0.70.0` | |
| `X-Stainless-OS` | `Linux` | |
| `X-Stainless-Arch` | `arm64` | |
| `X-Stainless-Runtime` | `node` | |
| `X-Stainless-Runtime-Version` | `v24.13.0` | |
| `X-Stainless-Retry-Count` | `0` | |
| `X-Stainless-Timeout` | `600` | |
| `X-App` | `cli` | |
| `Anthropic-Dangerous-Direct-Browser-Access` | `true` | |
| 系统提示前缀 | `You are Claude Code, Anthropic's official CLI for Claude.` | **不能尾随空白/换行**，拼接时另加 `\n\n` |
| 计费 attribution block | `cc_version=2.1.92.{fingerprint}` | fingerprint 算法见 `gateway_service.go` |
| URL | `https://api.anthropic.com/v1/messages?beta=true` + `…/count_tokens?beta=true` | |
| `cache_control` 上限 | 4 个 block | |
| `cache_control` TTL 默认 | `5m`（避免浪费 1h 额度） | |

**Mimicry 完整 beta 列表（OAuth, 非 Haiku）**：

```
claude-code-20250219,
oauth-2025-04-20,
interleaved-thinking-2025-05-14,
prompt-caching-scope-2026-01-05,
effort-2025-11-24,
context-management-2025-06-27,
extended-cache-ttl-2025-04-11
```

**模型 ID 映射**：

| 短名（对外接口可用） | 真实 ID（OAuth 必须用） |
|---------------------|------------------------|
| `claude-sonnet-4-5` | `claude-sonnet-4-5-20250929` |
| `claude-opus-4-5` | `claude-opus-4-5-20251101` |
| `claude-haiku-4-5` | `claude-haiku-4-5-20251001` |

**🔴 红线**：剥离过程中绝不能动 `internal/pkg/claude/`，绝不能动 `gateway_service.go` 里跟这些头相关的代码。

### 1.2 OpenAI Codex CLI OAuth（🔴）

来源：sub2api `internal/pkg/openai/`

| 字段 | 值 |
|------|-----|
| ClientID | `app_EMoamEEZ73f0CkXaXp7hrann` |
| AuthorizeURL | `https://auth.openai.com/oauth/authorize` |
| TokenURL | `https://auth.openai.com/oauth/token` |
| RedirectURI 默认 | `http://localhost:1455/auth/callback` |
| Scopes（首次） | `openid profile email offline_access` |
| Scopes（refresh） | `openid profile email`（**不带 offline_access**） |
| PKCE | S256 |
| Sticky session TTL | 1h |

**🔴 红线**：

- Refresh token 时**必须移除** `offline_access`，否则 token endpoint 返回 400
- `instructions.txt`（Codex CLI 的 system prompt 模板）必须 1:1 复刻，**不能在中间插自己的 system**
- WebSocket 路径 `/v1/responses` GET 升级是 Codex CLI 关键路径，不能漏

### 1.3 Antigravity OAuth（🔴）

来源：sub2api `internal/pkg/antigravity/oauth.go`

| 字段 | 值 |
|------|-----|
| ClientID | `1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com` |
| ClientSecret 默认 | `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` |
| ClientSecret ENV 覆盖 | `ANTIGRAVITY_OAUTH_CLIENT_SECRET` |
| User-Agent 版本 | `1.23.2`，可 ENV `ANTIGRAVITY_USER_AGENT_VERSION` 覆盖；**必须符合 `\d+\.\d+\.\d+`** |
| RedirectURI | `http://localhost:8085/callback` |
| Scopes | `cloud-platform` + `userinfo.email` + `userinfo.profile` + `cclog` + `experimentsandconfigs`（5 个 scope） |
| API Base（prod） | `cloudcode-pa.googleapis.com` |
| API Base（daily sandbox） | `daily-cloudcode-pa.sandbox.googleapis.com` |
| URLAvailabilityTTL | 5min |

**🔴 红线**：

- ClientSecret 明文在仓库里，Google 任何时候可 revoke
- **运行时必须优先读 ENV**，仓库里的值仅作 fallback
- 启动时**警告**用户该 secret 是临时的
- 双 base URL 必须保留，并实现 5min 可用性冷却

### 1.4 TLS 指纹（🔴）

来源：sub2api `internal/pkg/tlsfingerprint/`

- OpenAI / Anthropic / Google 都做 JA3/JA4 + HTTP/2 settings 反爬
- **绝对不能改成默认 Go net/http**（默认 ClientHello 一秒被识别）
- 必须保留 `tlsfingerprint/dialer.go` 整套逻辑
- 模拟的目标：Node.js v24.13 / Chrome / Codex CLI（按账号类型切换）

**🟡 严重**：seller-relay 跑在卖家本地，如果卖家用了代理（proxy chain），代理本身可能改 TLS 指纹，覆盖你的伪装。文档里要提醒卖家：用 SOCKS5 代理 OK，**不要用 HTTPS proxy**。

### 1.5 anthropic-beta 顺序敏感（🔴）

抓包顺序敏感。sub2api `FullClaudeCodeMimicryBetas()` 返回的字符串必须**原样拼接**，不要在中间排序、去重、加空格。

错误写法（会被降级）：

```go
betas := []string{"claude-code-20250219", "oauth-2025-04-20"}
sort.Strings(betas)  // ❌ 不要排序
header := strings.Join(betas, ",")
```

正确写法：

```go
// 直接用 sub2api 提供的常量字符串
header := claude.FullClaudeCodeMimicryBetas()
```

---

## 2. 计费与计量（🔴）

### 2.1 Sticky Session 切账户的计费陷阱

**场景**：买家用同一个 endpoint 发了很多请求，sub2api 因为某账户被风控自动切到下一个账户。

**陷阱**：切到新账户时如果不做特殊处理，新账户的 prompt 缓存为空，整个 prompt 会被当作 `input_tokens` 重新计费 —— 但买家已经为前面那些 tokens 付过钱了。

**正确做法**：sub2api 已经实现了 `ForceCacheBillingContextKey`，切账户时把 `input_tokens` 转为 `cache_read_input_tokens`。**这段逻辑在剥离过程中绝对不能删**。

**对 AI Bazaar 的影响**：state channel ticket 累计 `cumulative_tokens` 时必须按"折算后"的 tokens 算，不是原始 input_tokens。具体公式：

```
billable_tokens = output_tokens
                + input_tokens（首次）
                + cache_read_input_tokens × 0.1（缓存读取打 10% 折）
                + cache_creation_input_tokens × 1.25（缓存创建加 25%）
```

> 这套折扣比例**应当与上游一致**，否则卖家可能亏钱。具体数字以 Anthropic / OpenAI 公布的为准，写到代码常量里时要加 comment 标注来源。

### 2.2 流式响应的 usage 字段

**场景**：SSE 流式响应里，`usage` 字段往往在最后一个 chunk 里。

**陷阱**：如果只在响应结束才统计 usage，那么客户端断连时这次请求会"漏计"。

**正确做法**：sub2api 用 `usage_record_worker_pool` 在流过程中持续记录，结束时 finalize。剥离时这个 pool 不能删。

### 2.3 ticket sequence violation（🔴）

**场景**：买家本地维护 ticket seq，卖家本地也维护。

**陷阱**：买家可能"重放"旧 ticket 让卖家以为 cumulative 减少了。

**正确做法**：卖家保留收到的**最大** seq 的 ticket，任何 seq 不大于已有最大值的 ticket 直接拒绝（error code 4004）。

---

## 3. 密码学陷阱（🔴）

### 3.1 不要自己实现密码学

- Ed25519：用 `ed25519-dalek`，**不要用** `sodiumoxide` / `ring` 的 ed25519（API 不同，容易引入 bug）
- X25519：用 `x25519-dalek`
- ChaCha20-Poly1305：用 `chacha20poly1305` crate
- HKDF：用 `hkdf` crate
- 不要用 `secp256k1` 做身份签名（留给链层用）

### 3.2 nonce 必须真随机

```rust
use rand::rngs::OsRng;
let mut nonce = [0u8; 32];
OsRng.fill_bytes(&mut nonce);  // ✅
```

❌ 不要用 `thread_rng()` 做密码学 nonce（足够安全但 OsRng 更明显）
❌ 不要用时间戳作为 nonce（可预测，灾难）

### 3.3 私钥不能进日志

```rust
use zeroize::Zeroize;

#[derive(Zeroize)]
#[zeroize(drop)]
struct PrivateKey([u8; 32]);

impl std::fmt::Debug for PrivateKey {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "PrivateKey(redacted)")
    }
}
```

❌ 不要 derive `Debug` 在含私钥的结构上

### 3.4 助记词的存储

- 助记词只在两个时机出现：**生成时**显示给用户记录、**导入时**用户输入
- **绝不**写入磁盘
- **绝不**写入剪贴板
- 导入后立即派生密钥并存入 keychain，助记词内存擦除

### 3.5 commit-reveal 的 commitment 格式

按 PROTOCOL.md §3.3：

```
commitment = BLAKE3(
    CBOR.encode({
        "req_id": <bytes from req_id UUID>,
        "seller_fp": <20-byte fingerprint>,
        "price_micro_usdc": <u64>,
        "nonce": <32 random bytes>,
    })
)
```

**🔴 陷阱**：
- CBOR 必须用 **canonical 形式**（map keys sorted）
- 如果用 serde_cbor / ciborium，必须显式开 canonical mode
- Rust 和 Go 实现必须用同一组测试向量验证

---

## 4. GitHub 同步陷阱（🟡）

### 4.1 多 fork 一致性

**场景**：买家从 3 个 fork pull 数据，其中一个 fork 有滞后 / 被攻击者推了假数据。

**正确做法**：
- 同一文件路径，按 `min_quorum`（默认 2）个 fork 一致才采信
- 不一致的标 disputed，UI 提示

### 4.2 PR 合并 race

**场景**：两个买家同时给同一个卖家发请求，bot 同时 merge 两个 PR。

**陷阱**：如果不限流，bot 可能合并恶意 PR。

**正确做法**：
- bot 限流：同一 fp 每天最多 100 次写入
- bot 严格校验签名 + schema 后才 merge
- 客户端不依赖单一 fork

### 4.3 时间戳同步

不同客户端时钟可能漂移。建议：
- 拒绝 timestamp 超过 +5min 未来的消息（防止恶意未来戳）
- 接受 timestamp 在过去 24h 内的消息（容忍时钟回退）
- **不要**用 NTP 服务器作为信任根（攻击面）

---

## 5. 资金陷阱（🔴）

### 5.1 HTLC timelock 设置

**陷阱**：
- timelock 太短：卖家来不及 claim，资金被白白退回买家
- timelock 太长：买家资金长期锁住，体验差

**建议**：
- 小额（< $10）：timelock = 1 小时
- 中额（$10-$100）：timelock = 6 小时
- 大额（> $100）：timelock = 24 小时
- 单笔上限：v0 阶段建议 $50（拆单）

### 5.2 preimage 泄漏

**场景**：卖家用 preimage 取款时，链上交易公开 preimage。

**陷阱**：如果一个 preimage 被多个 HTLC 用，第一个 claim 后其他人也能用同一 preimage claim 别人的 HTLC。

**正确做法**：每个 HTLC 用独立的 preimage（OsRng 32 bytes）。

### 5.3 重入攻击

HTLC 合约必须用 **checks-effects-interactions** 模式 + reentrancy guard。OpenZeppelin 的 ReentrancyGuard 一定要用。

### 5.4 USDC 黑名单地址

**场景**：USDC 由 Circle 控制，可以冻结地址。如果卖家被冻结，卖家无法取款。

**陷阱**：买家钱被锁在 HTLC，卖家又取不出，资金死锁。

**正确做法**：
- HTLC 必须始终允许 timelock 到期后买家退款
- 买家退款路径不能依赖卖家任何操作
- 文档里告知用户 USDC 冻结风险

### 5.5 gas 余额

卖家 claim 需要 ETH 付 gas。如果卖家钱包没有 ETH，无法取款。

**正确做法**：buyer-cli 在创建 HTLC 时**额外打 0.0001 ETH 给卖家做 gas**（"gas dust"）。

---

## 6. Go-Rust IPC 陷阱（🟡）

### 6.1 socket 文件权限

```go
os.Chmod(socketPath, 0600)  // ✅ 仅本用户读写
```

❌ 不要 `0666`（任何用户可读）

### 6.2 大消息分帧

NDJSON 一行一条，但是要给 buffer size 设上限（建议 64 KB）。超过的拒绝。

### 6.3 JSON 兼容

- Rust serde + Go json/encoding 对数字类型不一致：
  - Go `int64` → JSON number；Rust 反序列化为 `u64` 没问题
  - Rust `f64` 大整数 → 精度丢失；**统一用字符串表示大金额**或拆成 `(high, low)`
- Rust `Option<T>::None` → `null`；Go `*T = nil` → `null`，但 Go 默认是 omit；需要在 struct 字段加 `,omitempty` 控制

---

## 7. sub2api 内部坑（🟡）

来源：W1 的 sub2api 分析报告

### 7.1 wire 删 service 后报循环依赖

**场景**：删除某 service 后，wire generate 报另一个保留 service 找不到 provider。

**正确做法**：
1. 把保留 service 的依赖改为接口 + 注入 noop 实现
2. 千万**不要**为了让 wire 通过把删掉的 service 加回来

### 7.2 ent generate 后 predicate / hook 报错

**场景**：删了某 schema，predicate 包还引用它。

**正确做法**：
- 删 schema 后**必须**重跑 `go generate ./ent/...`
- 删的同时确保没有其他 schema 通过 Edge 引用它

### 7.3 security_secret_bootstrap

**场景**：sub2api 首次启动写 JWT/AES 密钥到 DB。

**陷阱**：剥离后改成 ENV 注入，但已经部署的实例升级会丢密钥。

**正确做法**：
- 优先读 ENV
- ENV 未设 → 读 DB 旧值
- DB 没有 → 生成新值写 DB（兼容老部署）

### 7.4 aes_encryptor 统一密钥

**场景**：sub2api 用统一 AES key 加密所有 OAuth credential。

**陷阱**：seller-relay 单卖家场景下，应该每卖家本地一个 key（存 keychain），但如果暴力替换会导致老数据读不出。

**正确做法**：
- v0 阶段：保留统一 key 但从 keychain 读，不写代码常量
- v1 阶段：迁移到 per-account encryption

### 7.5 OpsErrorLoggerMiddleware

**场景**：handler 注册时挂载这个中间件。

**陷阱**：删 ops_service 后这个中间件没了，gateway 路由会报错。

**正确做法**：保留中间件的"裸版本"，去掉对 ops_service 的依赖，只做 endpoint normalize + 错误日志。

---

## 8. 隐私陷阱（🔴）

### 8.1 卖家能看到买家 prompt

**事实**：seller-relay 是 HTTP 代理，**它一定看得到 plaintext prompt**。这是物理上限。

**正确做法**：
- 文档明确告知买家：敏感数据不要走二级市场
- 卖家在 listing 中声明 `logs_request_content: false`
- 违反者通过声誉系统惩罚

❌ **不要** 宣称 "端到端加密 prompt"（除非用 TEE，v0 不做）

### 8.2 元数据泄漏

即使加密 prompt，元数据（时间、tokens 数、模型）仍然可见。

**v0 接受**：
- 卖家可以看到买家的 fingerprint、请求时间、token 数
- 买家可以选 carrier（v1 才有）做洋葱路由隐藏 fingerprint

### 8.3 日志清洗

```rust
tracing::info!(
    target = "request",
    buyer_fp = %fp_hex_with_redact(&fp),  // ✅ 仅记录前 8 字符
    tokens = used,
    "request handled"
);
```

❌ 不要直接 `?buyer_fp` 把完整 fp 打到日志

❌ 不要在日志里出现 prompt content / response content / api key

---

## 9. 合规与法律陷阱（🟡）

### 9.1 KYC

- 链上钱包**不需要** KYC
- 但用户在中心化交易所买 USDC 时**会**触发 KYC
- 我们的产品**不做**法币入金，避免 MSB 牌照问题

### 9.2 ToS 措辞

❌ 不要写：
- "use anyone's Claude subscription"
- "bypass OpenAI subscription"
- "resell API access"

✅ 可以写：
- "share unused API quota with consent"
- "P2P API request relay"
- "decentralized capacity exchange"

### 9.3 主仓库定位

- 主仓库放在中立组织（不要个人）
- 不放官方域名、官方下载、客服联系方式
- README 只描述协议，不描述如何"用便宜的 Claude"
- 客户端代码里不内置任何 fork URL（用户自带 manifest）

---

## 10. 性能陷阱（🟢）

### 10.1 SSE 转发的 buffer

- 不要 buffer 整个响应（破坏流式体验）
- 每 chunk 直接 flush
- sub2api 已经处理好，剥离时不要动 `sse_*` 文件

### 10.2 GitHub API rate limit

- 未认证：60 req/h（不够用）
- 认证：5000 req/h（够用）
- 建议每个客户端配置 PAT，从 keychain 读

### 10.3 Database 锁

SQLite WAL mode 仍可能写锁。如果 seller-relay 突然 freeze 几秒，检查是否有长事务。

---

## 11. 自检清单（每个 milestone 跑一遍）

完成 W1-W2（sub2api strip）后检查：

- [ ] `internal/pkg/claude/constants.go` 内容未改
- [ ] `internal/pkg/openai/` OAuth 常量未改
- [ ] `internal/pkg/antigravity/oauth.go` 常量未改
- [ ] `internal/pkg/tlsfingerprint/` 完整保留
- [ ] sticky session 切账户的 ForceCacheBilling 逻辑保留
- [ ] 四个错误计数器（401/403/429/internal500）保留
- [ ] usage_record_worker_pool 保留
- [ ] PostgreSQL / Redis 依赖删干净
- [ ] 没有把 ClientSecret 提交进代码（验证：`git grep "GOCSPX"` 仅在原文件出现）
- [ ] `apicompat/` 完整保留（即使 v0 不开启跨协议）

完成 W4（公钥签名）后检查：

- [ ] 私钥所有结构 Zeroize + Drop
- [ ] Debug impl 不暴露私钥
- [ ] nonce 全部用 OsRng
- [ ] 助记词不进磁盘 / 剪贴板
- [ ] commit-reveal 测试向量与 Rust/Go 互验证通过

完成 W7（commit-reveal）后检查：

- [ ] 重复 commit 拒绝
- [ ] reveal 解密后 commitment 验证失败 → 声誉惩罚
- [ ] 过期 reveal 不接收
- [ ] all_commitments 在 tx 中包含所有参与者

完成 W8（HTLC）后检查：

- [ ] HTLC 合约用 ReentrancyGuard
- [ ] preimage 不复用
- [ ] 买家 refund 路径不依赖卖家
- [ ] ticket seq 不可回放
- [ ] gas dust 转账成功
