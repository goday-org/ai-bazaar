# 术语对照表（GLOSSARY.md）

> 整个代码库统一使用本表的英文标识符。中文仅在文档 / 注释中使用。

---

## 角色

| 中文 | 英文 | 标识符 | 说明 |
|------|------|--------|------|
| 买家 | Buyer | `buyer` | 想用 API 的人 |
| 卖家 | Seller | `seller` | 有富余订阅的人 |
| 中继节点 | Carrier | `carrier` | 洋葱路由跳板（v1） |
| 审查者 / 评价者 | Reviewer（评价场景） | `reviewer` | 给某个 tx 评价的人 |

> 注：本项目的"reviewer"在两个场景出现：
> - **代码 review**：开发流程中的 reviewer（人）
> - **声誉 review**：交易后互评的角色
> 代码里第二种用 `Rater` 避免混淆。

---

## 协议对象

| 中文 | 英文 | 标识符 | GitHub 路径 |
|------|------|--------|-------------|
| 卖家挂牌 | Listing | `listing` | `sellers/<fp>.json` |
| 买家需求 | Request | `request` | `requests/<fp>/<req_id>.json` |
| 报价承诺 | Commit | `commit` | `bids/<req_id>/<seller_fp>.commit` |
| 报价揭示 | Reveal | `reveal` | `bids/<req_id>/<seller_fp>.reveal` |
| 成交记录 | Transaction | `tx` | `tx/<yyyy-mm>/<tx_id>.json` |
| 计量凭证 | Ticket | `ticket` | （不上传） |
| 声誉评价 | Rating | `rating` | `reputation/<fp>/<rev_id>.json` |
| 黑名单声明 | Revocation | `revocation` | `revocations/<fp>.json` |
| 镜像清单 | Manifest | `manifest` | `manifest.json` |

---

## 密码学

| 中文 | 英文 | 标识符 |
|------|------|--------|
| 公钥 | Public Key | `pubkey` |
| 私钥 | Private Key | `seckey` (避免与 SK 缩写混淆) |
| 公钥指纹 | Fingerprint | `fp` |
| 一次性会话密钥 | Ephemeral Key | `epk` / `esk` |
| 承诺哈希 | Commitment | `commitment` |
| 随机 nonce | Nonce | `nonce` |
| 签名 | Signature | `signature` |

---

## 服务标识

| 含义 | 标识符 | 格式 | 示例 |
|------|--------|------|------|
| 服务 ID | `service_id` | `<vendor>.<model_family>[-<variant>][@<version>]` | `anthropic.claude-sonnet-4.5` |
| 厂商 | `vendor` | 小写 | `anthropic`, `openai`, `google` |
| 模型族 | `model_family` | 小写 + 连字符 | `claude-sonnet-4.5` |

完整列表见 [`PROTOCOL.md §3.7`](PROTOCOL.md#37-服务命名service-ids)。

---

## 资金 / 计量

| 中文 | 英文 | 标识符 | 单位 |
|------|------|--------|------|
| 价格 | Price | `price_micro_usdc` | 微 USDC（1 USDC = 1,000,000） |
| 单次成交价 | Final Price | `final_price_micro_usdc` | 同上 |
| 上限价 | Max Price | `max_price_micro_usdc` | 同上 |
| Token 数量 | Tokens | `tokens` | 整数 |
| 累计 Token | Cumulative Tokens | `cumulative_tokens` | 整数 |
| 累计金额 | Cumulative Price | `cumulative_price_micro_usdc` | 微 USDC |
| Ticket 序号 | Sequence | `seq` | u64 单调递增 |

> **价格单位规则**：协议层只用整数微 USDC，避免浮点精度问题。UI 层显示时再除以 1,000,000。

---

## 状态机

### Request 状态

| 中文 | 英文 / 标识符 |
|------|--------------|
| 草稿 | `DRAFT` |
| 已发布 | `PUBLISHED` |
| Commit 截止 | `COMMIT_CLOSED` |
| Reveal 截止 | `REVEAL_CLOSED` |
| 选定中标方 | `WINNER_SELECTED` |
| 链上托管已开 | `HTLC_OPEN` |
| 服务中 | `IN_PROGRESS` |
| 已完成 | `COMPLETED` |
| 已退款 | `REFUNDED` |
| 已撤销 | `CANCELLED` |
| 已过期（无有效 reveal） | `EXPIRED` |
| 争议中 | `DISPUTED` |

### Bid 状态（卖家视角）

| 中文 | 英文 / 标识符 |
|------|--------------|
| 已看到需求 | `SEEN_REQUEST` |
| 已 commit | `COMMITTED` |
| 已 reveal | `REVEALED` |
| 已弃权（未按时 reveal） | `FORFEITED` |
| 中标 | `WON` |
| 落选 | `LOST` |
| 服务中 | `SERVING` |
| 已 claim | `CLAIMED` |
| 已超时 | `TIMED_OUT` |

---

## 错误码

完整列表见 [`PROTOCOL.md §7`](PROTOCOL.md#7-错误码)。

按范围分类：

- `1xxx` 协议错误（签名 / 时间戳 / schema）
- `2xxx` 竞价错误（deadline / 重复 commit / 价格越界）
- `3xxx` 执行错误（上游 API / 容量）
- `4xxx` 结算错误（HTLC / ticket）
- `5xxx` 系统错误（DB / 内部异常）

---

## 配置文件

| 路径 | 用途 |
|------|------|
| `~/.ai-bazaar/buyer.toml` | buyer-cli 配置 |
| `~/.ai-bazaar/seller.toml` | seller-relay + seller-ctl 配置 |
| `~/.ai-bazaar/manifest.json` | 镜像清单 |
| `~/.ai-bazaar/seller.db` | seller SQLite |
| `~/.ai-bazaar/buyer.db` | buyer 本地缓存 |
| `~/.ai-bazaar/buyer.sock` | buyer IPC socket |
| `~/.ai-bazaar/seller.sock` | seller IPC socket |
| `~/.ai-bazaar/keys/` | 密钥（默认是 keychain，此目录仅作 fallback） |
| `~/.ai-bazaar/logs/` | 日志 |

---

## 进程命名

| 二进制 | 语言 | 角色 |
|--------|------|------|
| `seller-relay` | Go | 卖家本地代理（API 转发） |
| `seller-ctl` | Rust | 卖家控制平面（签名 / 竞价 / GitHub） |
| `buyer-cli` | Rust | 买家无头客户端（OpenAI 兼容 endpoint） |
| `buyer-client` | Rust + Tauri | 买家桌面应用 |
| `registry-bot` | Rust | GitHub PR auto-merge bot |
| `ipc-probe` | Rust | 调试工具 |

---

## 缩写

| 缩写 | 全称 |
|------|------|
| ADR | Architecture Decision Record |
| AEAD | Authenticated Encryption with Associated Data |
| BIP39 | Bitcoin Improvement Proposal 39（助记词标准） |
| BLAKE3 | 第 3 代 BLAKE 哈希 |
| CBOR | Concise Binary Object Representation |
| ECDH | Elliptic Curve Diffie-Hellman |
| EVM | Ethereum Virtual Machine |
| FP | Fingerprint |
| HKDF | HMAC-based Key Derivation Function |
| HTLC | Hash Time-Locked Contract |
| IPC | Inter-Process Communication |
| JA3/JA4 | TLS Client Hello 指纹算法 |
| MSB | Money Services Business |
| PR | Pull Request |
| PSK | Pre-Shared Key |
| RPC | Remote Procedure Call |
| SBOM | Software Bill of Materials |
| SSE | Server-Sent Events |
| TEE | Trusted Execution Environment |
| ToS | Terms of Service |
| TUI | Terminal User Interface |
| WAL | Write-Ahead Log (SQLite) |

---

## 不要使用的词

以下词在代码、UI、文档中**禁止**出现：

- "resell" / "reselling" → 用 "share quota"
- "share account" → 用 "P2P relay"
- "bypass subscription" → 不要写这个意思
- "anonymous" + "untraceable" 一起出现 → 改写
- "evade" / "evading" → 改写
- "platform anti-fraud" 的中文"反风控" / "绕过风控" → 改写
- 任何 AI 平台官方名字 + "下游" / "黑产" 组合 → 改写

理由：法务与社区观感。详见 [`HANDOFF.md §2`](HANDOFF.md#2-法律与道德边界你必须理解的)。
