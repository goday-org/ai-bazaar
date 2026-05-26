# 协议规范（PROTOCOL.md）

> **本文档是契约**。修改前必须 PR。所有 Rust 和 Go 实现必须严格遵守。
> **当前版本**：`protocol_version: 0.1.0`

---

## 1. 标识符

### 1.1 身份（Identity）

每个参与者由一对 Ed25519 密钥定义。

- **PublicKey**：32 bytes，编码为 base64url（无 padding）
- **PublicKeyFingerprint**：BLAKE3-128 of PublicKey，编码为 base32（小写，无 padding），20 字符
  - **BLAKE3-128 的精确定义**：取 BLAKE3 标准 256-bit 输出的**前 16 字节**。
    伪代码：`fp_bytes = blake3::hash(pubkey).as_bytes()[..16]`。
    Rust 用 `blake3::hash(&pk).as_bytes()` 取前 16 字节；Go 用 `lukechampine.com/blake3` 的 `Sum256(pk)` 再切片。
    **不要用** `finalize_xof()` 读 16 字节（XOF 模式输出与截断模式不同）。
- **示例**：`abcdefghij1234567890`

### 1.2 路径与文件名

GitHub 仓库内所有路径用 fingerprint，**不要用 base64 公钥**（避免 `/` 和大小写问题）。

```
sellers/<fp>.json
requests/<fp>/<req_id>.json
bids/<req_id>/<seller_fp>.commit
bids/<req_id>/<seller_fp>.reveal
tx/<yyyy-mm>/<tx_id>.json
reputation/<fp>/<rev_id>.json
revocations/<fp>.json
manifest.json
```

### 1.3 ID 生成

- `req_id`：UUID v7（时间排序）
- `tx_id`：BLAKE3-128 of `req_id_bytes || winner_fp_bytes || final_price_u64_le`，base32
- `rev_id`：BLAKE3-128 of `tx_id_bytes || rater_fp_bytes`，base32

**BLAKE3-128 与 §1.1 同义**：取 256-bit 输出的前 16 字节。
**整数编码统一用 little-endian**（`final_price_u64_le` 表示 8-byte LE）。

---

## 2. 签名规范

### 2.1 签名算法

- **算法**：纯 Ed25519（RFC 8032）
- **哈希**：SHA-512（Ed25519 内置）
- **不要用** Ed25519ph / Ed25519ctx 变体

### 2.2 签名 payload

所有签名采用 **detached signature** 模式：

```
signature = Ed25519_Sign(
    private_key,
    canonical_json(message_without_signature_field)
)
```

**重要**：直接对 canonical_json 字节签名，**不要预先 hash**。Ed25519 内部已经做 SHA-512，再加一层 SHA-256 会偏离 RFC 8032，且与 Rust `ed25519-dalek` / Go `crypto/ed25519` 默认行为不一致，导致 Rust/Go 实现互不兼容。

**canonical_json 规则**（必须 Rust / Go 两边一致）：

1. 所有 object key 按 UTF-8 字节序排序
2. 无空格、无换行
3. 整数原样输出；**协议禁止用浮点**（价格全部用整数微单位 u64）
4. 字符串按 RFC 8259 escape，最小化转义（仅 `"`、`\`、`\b\f\n\r\t`、U+0000–U+001F 用 `\u00XX`，其余原样输出）
5. 数组保持声明顺序，不重排序

### 2.3 签名字段位置

每个消息顶层固定字段：

```json
{
  "pv": "0.1.0",
  "type": "...",
  "timestamp": "2026-05-26T10:30:00Z",
  "...具体字段...": "...",
  "signature": {
    "signer": "<base64url 32-byte pubkey>",
    "value": "<base64url 64-byte sig>"
  }
}
```

签名时**排除 `signature` 字段本身**，对剩余字段做 canonical_json 再签名。

### 2.4 时间戳

- 格式：RFC 3339，UTC，带 Z 后缀，无毫秒
- 验证规则：接收时若 `|now - timestamp| > 24h`，拒绝该消息
- 例外：`tx/*.json` 不受此限（要保留历史）

---

## 3. 消息类型

### 3.1 Seller Listing（卖家挂牌）

文件：`sellers/<fp>.json`

```json
{
  "pv": "0.1.0",
  "type": "listing",
  "timestamp": "2026-05-26T10:00:00Z",
  "seller_pubkey": "<base64url>",
  "services": [
    {
      "id": "anthropic.claude-sonnet-4.5",
      "capacity_per_day": 2000000,
      "min_unit_tokens": 1000,
      "max_unit_tokens": 100000
    }
  ],
  "endpoint_hints": [
    "tor://abcdefghij1234567890.onion:9000",
    "https://relay.example.org:9000"
  ],
  "noise_static_pubkey": "<base64url 32-byte X25519 pubkey>",
  "policies": {
    "log_retention_days": 0,
    "logs_request_content": false,
    "supports_streaming": true
  },
  "signature": { "signer": "...", "value": "..." }
}
```

**字段语义**：

- `services[].id`：见 §3.7 服务命名表
- `capacity_per_day`：上游剩余额度（tokens / day）。卖家自报，买家不信任，只作排序参考
- `noise_static_pubkey`：X25519 公钥，用于 Noise_IK 握手。可以与 seller_pubkey 不同
- `policies.logs_request_content`：诚信声明，违背则声誉惩罚

**生命周期**：
- 有效期 7 天（基于 timestamp）
- 卖家应每天刷新一次
- 过期的 listing 客户端忽略

### 3.2 Buyer Request（买家需求）

文件：`requests/<buyer_fp>/<req_id>.json`

```json
{
  "pv": "0.1.0",
  "type": "request",
  "timestamp": "2026-05-26T10:30:00Z",
  "req_id": "01998b6f-...",
  "buyer_pubkey": "<base64url ed25519>",
  "buyer_x25519": "<base64url x25519>",
  "service": "anthropic.claude-sonnet-4.5",
  "quantity_tokens": 100000,
  "max_price_micro_usdc": 5000000,
  "commit_deadline": "2026-05-26T10:35:00Z",
  "reveal_deadline": "2026-05-26T10:40:00Z",
  "settlement_chain": "base",
  "settlement_token": "USDC",
  "signature": { "signer": "...", "value": "..." }
}
```

**字段语义**：

- `quantity_tokens`：买家承诺最大购买量
- `max_price_micro_usdc`：买家愿出的最高总价（1 USDC = 1,000,000 微单位）。卖家报价高于此值自动失格
- `commit_deadline`：commit 阶段截止，超过不接收新 commit
- `reveal_deadline`：reveal 阶段截止，超过未 reveal 的 commit 失格 + 声誉惩罚

### 3.3 Commit（密封承诺）

文件：`bids/<req_id>/<seller_fp>.commit`

```json
{
  "pv": "0.1.0",
  "type": "commit",
  "timestamp": "2026-05-26T10:32:00Z",
  "req_id": "01998b6f-...",
  "seller_pubkey": "<base64url>",
  "commitment": "<base64url 32-byte BLAKE3 hash>",
  "signature": { "signer": "...", "value": "..." }
}
```

**commitment 计算**（Rust 和 Go 必须 byte-for-byte 一致）：

```
commitment = BLAKE3(
    canonical_cbor({
        "nonce":            <bstr, 32 bytes>,
        "price_micro_usdc": <uint, u64>,
        "req_id":           <bstr, 16 bytes from UUID>,
        "seller_fp":        <bstr, 16 bytes>,
    })
)
```

**重要**：
- BLAKE3 输出取**完整 32 字节**（不是 BLAKE3-128 的 16 字节；commitment 字段长度 = 32）
- CBOR map keys 按上方字典序排列；Rust 用 `ciborium::ser::into_writer` + 手动构造 BTreeMap，Go 用 `github.com/fxamacker/cbor/v2` 的 `core deterministic` mode
- `seller_fp` 是 PublicKeyFingerprint（16 bytes 原始字节，不是 base32 编码后的字符串）
- `req_id` 是 UUID 的 16 字节 raw（不是连字符字符串）

**约束**：
- 同一 seller 对同一 req_id 只能有一个 commit。重复 commit → 后到的拒绝，并视为试图作弊
- commit 必须在 `request.commit_deadline` 之前提交

### 3.4 Reveal（揭示）

文件：`bids/<req_id>/<seller_fp>.reveal`

```json
{
  "pv": "0.1.0",
  "type": "reveal",
  "timestamp": "2026-05-26T10:37:00Z",
  "req_id": "01998b6f-...",
  "seller_pubkey": "<base64url>",
  "encrypted_to_buyer": "<base64url ciphertext>",
  "signature": { "signer": "...", "value": "..." }
}
```

**encrypted_to_buyer 构造**：

1. 卖家与买家执行一次性 X25519 ECDH：
   - 卖家生成 ephemeral X25519 keypair `(esk, epk)`
   - `shared = X25519(esk, request.buyer_x25519)`
   - `key = HKDF-SHA256(shared, salt="ai-bazaar/reveal/v1", info=req_id || seller_fp, 32 bytes)`
2. 用 ChaCha20-Poly1305 加密 plaintext：

```json
{
    "price_micro_usdc": 4500000,
    "nonce": "<base64url 32-byte>",
    "delivery_endpoint": "https://relay.example.org:9000",
    "noise_static_pubkey": "<base64url x25519>"
}
```

3. 组装密文：

```
encrypted_to_buyer = base64url(
    epk_32_bytes
    || chacha20poly1305_nonce_12_bytes
    || ciphertext_with_16_byte_tag
)
```

**约束**：
- 必须在 `request.reveal_deadline` 之前提交
- 解密后必须用**与 §3.3 完全相同的 canonical_cbor 构造**重算 commitment 验证（即把 `(price_micro_usdc, nonce, req_id, seller_fp)` 按 §3.3 公式重算 BLAKE3 32 字节，与 commit 文件里的 `commitment` 字段比对）。
  ❌ **不要**用其他形式（如直接拼接 `price || nonce || ...`）——必须走 CBOR
- 验证失败 → 视为作弊，声誉惩罚 + error code `2005 commitment_mismatch`

### 3.5 Transaction（成交记录）

文件：`tx/<yyyy-mm>/<tx_id>.json`

```json
{
  "pv": "0.1.0",
  "type": "tx",
  "timestamp": "2026-05-26T10:41:00Z",
  "tx_id": "<base32>",
  "req_id": "01998b6f-...",
  "buyer_pubkey": "<base64url>",
  "winner_pubkey": "<base64url>",
  "winning_bid_micro_usdc": 4200000,
  "final_price_micro_usdc": 4500000,
  "quantity_tokens": 100000,
  "all_commitments": [
    {"seller_fp": "abc...", "commitment": "..."},
    {"seller_fp": "def...", "commitment": "..."}
  ],
  "settlement": {
    "chain": "base",
    "htlc_address": "0x...",
    "hashlock": "0x...",
    "timelock": 1701234567
  },
  "buyer_signature": "<base64url>",
  "seller_signature": "<base64url>"
}
```

**Vickrey（second-price sealed-bid）拍卖规则**：

| 场景 | `winner_pubkey` | `winning_bid_micro_usdc` | `final_price_micro_usdc`（实际付款） |
|------|----------------|--------------------------|---------------------------------------|
| ≥ 2 个有效 reveal | 最低价 reveal 的 seller | 该 seller 的 reveal 价 | **第二低**有效 reveal 价 |
| 仅 1 个有效 reveal | 该 seller | 该 seller 的 reveal 价 | `request.max_price_micro_usdc`（reservation price 兜底） |
| 0 个有效 reveal | —— | —— | request 进入 `EXPIRED`，不生成 tx |
| 最低价并列（含 N 路) | 用 `BLAKE3(req_id_bytes ‖ sorted_seller_fps_bytes)` 的最后一个字节 mod N 选 winner（**确定性**：任何买家本地算出同一个胜者） | 该 seller 的 reveal 价 | 第二低有效 reveal 价（若所有人并列同一最低价，等于该价） |

**为什么是 Vickrey**：让卖家"如实报真实成本"成为占优策略。卖家不需要去猜对手出多少。

**字段语义**：
- `winning_bid_micro_usdc`：winner 自己的 reveal 价（用于审计 + 落选者验证）
- `final_price_micro_usdc`：winner 实际收到的款（state channel ticket 计费基准）
- `final_price_micro_usdc ≥ winning_bid_micro_usdc` 始终成立（second-price 必然 ≥ first-price）

**双签名**：
- buyer 先签所有字段（除两个 signature），把 `buyer_signature` 填上
- 把 tx 通过 Noise 信道给 winner
- winner 验证后追加 `seller_signature`
- 任一方签完即可 PR 到 GitHub

**all_commitments**：包含**所有**参与 commit 的 seller 哈希（含未 reveal / reveal 失败者）。让落选者事后能验证：
1. 自己的 commitment 在列表里 → 没被买家无声忽略
2. 自己的 reveal 价 ≥ `final_price_micro_usdc` → 买家挑 winner 没作弊（落选者拿到 tx 后用自己的 nonce 重算 commitment 比对，再核对中标价合理）

### 3.6 State Channel Ticket（计量凭证）

**不上传 GitHub**。仅在双方本地累积 + 卖家终结时上链。

```json
{
  "pv": "0.1.0",
  "type": "ticket",
  "tx_id": "<base32>",
  "seq": 42,
  "cumulative_tokens": 87000,
  "cumulative_price_micro_usdc": 3915000,
  "timestamp": "2026-05-26T10:55:23Z",
  "buyer_signature": "<base64url>"
}
```

**约束**：
- `seq` 单调递增
- `cumulative_*` 单调递增
- `cumulative_price_micro_usdc / cumulative_tokens` 必须等于 tx 里的 `final_price_micro_usdc / quantity_tokens`（避免买家偷改单价）
- `cumulative_tokens <= tx.quantity_tokens`
- 卖家每收到一张 ticket 都验证签名，存盘
- 上链时只提交"序号最大"的那张

**传输信道（必读）**：

ticket **不通过 GitHub** 传递；它走买卖双方建立的 Noise 信道。具体：

1. winner 完成 §3.5 双签名后，使用 listing 中声明的 `noise_static_pubkey` 在 `endpoint_hints` 给出的某个地址建立 Noise_IK session
2. buyer-cli 把所有上游 API 请求经过该 session 转发到 seller-relay
3. **每次请求成功（上游返回完整响应 + usage）后**，buyer-cli 在同一 session 上 piggyback 一张新的 ticket（`seq = 上一个 + 1`，`cumulative_*` 累加）
4. seller-relay 校验签名 + 单调性后存盘；任一项不通过返回 error code `4003 ticket_signature_invalid` / `4004 ticket_sequence_violation`，并停止转发后续请求
5. 服务结束（quantity 用尽 / buyer 主动关闭 / timelock 临近）→ seller 拿 `seq` 最大的那张 ticket 上链 claim

**如果 Noise session 中断**：buyer-cli 重连后，必须从"卖家上一次 ack 的 seq" + 1 继续；不能 fork seq 序列。

### 3.7 服务命名（Service IDs）

**协议层 ID** 格式：`<vendor>.<model_family>[-<variant>]`，仅用小写连字符与点。

| 协议 ID（出现在 listing / request / IPC） | 上游真实 model ID（仅 seller-relay 内部用） | 上游 endpoint |
|----|------|---|
| `anthropic.claude-sonnet-4-5` | `claude-sonnet-4-5-20250929` | `api.anthropic.com` (OAuth subscription) |
| `anthropic.claude-opus-4-5` | `claude-opus-4-5-20251101` | 同上 |
| `anthropic.claude-haiku-4-5` | `claude-haiku-4-5-20251001` | 同上 |
| `openai.gpt-5` | `gpt-5`（占位，实际跟 Codex CLI 漂移） | `api.openai.com` (Codex OAuth) |
| `openai.o5` | `o5` | 同上 |
| `google.gemini-3-pro` | `gemini-3-pro-preview-XX` | `generativelanguage.googleapis.com` |
| `google.gemini-3-flash` | `gemini-3-flash-preview-XX` | 同上 |
| `google.antigravity-gemini-3` | 由 Antigravity 网关解析 | `cloudcode-pa.googleapis.com` |
| `google.antigravity-claude-opus-4-5` | 由 Antigravity 网关解析 | 同上 |

**命名规则**：
- 协议层全部用 **连字符**（`claude-sonnet-4-5`），不用点号或下划线
- 上游真实 ID（含日期戳）的漂移由 seller-relay `internal/pkg/<vendor>/` 包内部映射
- 买家 / 买家 UI / 买家 CLI **永远只**写协议层 ID
- 协议层 ID 一旦发布不能改；上游 ID 漂移由 seller-relay 适配（不影响协议）

> 协议层 ID 表的更新走 `proto:` 类型 PR，必须在 §10 修改流程下记录。

---

## 4. 状态机

### 4.1 Request 状态机（买家视角）

```
DRAFT
  │ 用户 commit
  ▼
PUBLISHED ─────────► CANCELLED（用户主动撤销）
  │
  │ commit_deadline 到达
  ▼
COMMIT_CLOSED
  │
  │ reveal_deadline 到达
  ▼
REVEAL_CLOSED
  │
  │ 至少一个有效 reveal
  ▼
WINNER_SELECTED ───► EXPIRED（无 valid reveal）
  │
  │ 链上 HTLC 创建
  ▼
HTLC_OPEN
  │
  │ 卖家开始服务 / 累积 tickets
  ▼
IN_PROGRESS
  │
  ├──► COMPLETED（卖家成功 claim）
  ├──► REFUNDED（timelock 到期，买家取回）
  └──► DISPUTED（异常路径，未来定义）
```

### 4.2 Seller Bid 状态机（卖家视角）

```
SEEN_REQUEST
  │ 决定竞标
  ▼
COMMITTED
  │ reveal_deadline 之前
  ▼
REVEALED ─────► FORFEITED（未在 reveal_deadline 前 reveal）
  │
  │ tx 公布
  ▼
WON / LOST
  │ （仅 WON 路径继续）
  │ Noise 握手
  ▼
SERVING
  │ 收 tickets
  ▼
CLAIMED / TIMED_OUT
```

---

## 5. GitHub 同步层

### 5.1 Manifest

每个客户端启动时读取 `manifest.json`（用户提供或默认）：

```json
{
  "pv": "0.1.0",
  "registry_forks": [
    "https://github.com/org-a/ai-bazaar-registry",
    "https://github.com/org-b/ai-bazaar-registry",
    "https://github.com/org-c/ai-bazaar-registry"
  ],
  "ipfs_backup_cid": "bafy...",
  "min_quorum": 2
}
```

### 5.2 视图合并

客户端定期（建议 30s）从所有 forks pull，合并视图：

- 同一文件路径在不同 fork 出现，取 **min_quorum** 个 fork 一致的版本
- 不一致的标记为 "disputed"，UI 提示用户
- 任意 fork 缺失文件视为 "未投票"，不影响多数判断

### 5.3 PR 自动 merge

每个 fork 应配一个 bot（`tools/registry-bot/`，由仓库 maintainer 各自跑）：

- 接受任何 PR
- 检查每个新增文件：
  1. 路径符合 §1.2 规则
  2. 文件名匹配签名者 fp（如 `sellers/<fp>.json` 的 signer 必须等于 fp）
  3. 签名有效
  4. 时间戳新鲜
  5. JSON schema 通过
- 通过 → auto-merge
- 失败 → close + comment 说明原因

### 5.4 限流

- 同一 pubkey 每天最多 100 次写入（防滥用）
- bot 维护一个本地黑名单，命中即拒
- 黑名单仅作用于本 fork，不同 fork 可有不同策略

---

## 6. Control Plane（IPC 协议）

### 6.1 Transport

- **buyer**：buyer-cli ↔ buyer-tauri 通过 Unix Domain Socket（Windows 用 Named Pipe），路径 `~/.ai-bazaar/buyer.sock`
- **seller**：seller-ctl ↔ seller-relay 通过 Unix Domain Socket，路径 `~/.ai-bazaar/seller.sock`

### 6.2 编码

JSON-RPC 2.0，每条消息一行（NDJSON）。

### 6.3 Seller IPC 方法

```
RPC OpenEndpoint
  params: {
    buyer_pubkey: string,
    service_id: string,
    quota_tokens: u64,
    expires_at: ISO8601
  }
  result: {
    endpoint_id: string (uuid),
    local_url: string,
    api_key: string,
    protocol_compat: ["anthropic", "openai", "gemini"]
  }
  // 字段语义：
  //   local_url:  seller-relay 在卖家本机为这个 endpoint 暴露的 HTTP 监听地址
  //               (例如 "http://127.0.0.1:11401")。这是 seller-relay 与
  //               seller-ctl 之间的 internal URL；不会直接暴露给 buyer。
  //               buyer 通过 seller listing 中的 endpoint_hints / Noise
  //               session 触达；seller-ctl 把 Noise 解出的请求转发到这个
  //               local_url。
  //   api_key:    40-char hex 字符串（= 20 bytes 随机），seller-relay 用来
  //               鉴权这个 endpoint 上的请求。仅在 seller 本机有效。

RPC CloseEndpoint
  params: { endpoint_id: string }
  result: { closed_at: ISO8601 }

RPC GetUsage
  params: { endpoint_id: string }
  result: {
    tokens_consumed: u64,
    requests_count: u64,
    last_request_at: ISO8601
  }

RPC ListAccounts
  params: {}
  result: {
    accounts: [{
      platform: string,
      account_id: string,
      status: "healthy" | "rate_limited" | "banned" | "expired",
      capacity_remaining: u64
    }]
  }

NOTIFY EndpointEvent (server → client)
  params: {
    endpoint_id: string,
    event: "request" | "error" | "quota_warning",
    payload: object
  }
```

### 6.4 Buyer IPC 方法

```
RPC ListMarketplace
  params: { service_filter?: string }
  result: { listings: [Listing] }

RPC SubmitRequest
  params: { request: Request }
  result: { req_id: string }

RPC GetRequestStatus
  params: { req_id: string }
  result: { state: RequestState, bids_count: u64, winner?: PublicKey }

RPC AcceptWinner
  params: { req_id: string, winner_pubkey: string }
  result: { tx_id: string, htlc_address: string }

RPC GetActiveEndpoints
  params: {}
  result: {
    endpoints: [{
      endpoint_id: string,
      local_url: string,
      service_id: string,
      tokens_used: u64,
      tokens_remaining: u64,
      seller_fp: string
    }]
  }
```

---

## 7. 错误码

所有 RPC / HTTP 错误使用统一 code（u32）：

```
1xxx  协议错误
  1001  invalid_signature
  1002  expired_timestamp
  1003  unknown_message_type
  1004  schema_validation_failed
  1005  protocol_version_mismatch

2xxx  竞价错误
  2001  request_not_found
  2002  commit_deadline_passed
  2003  reveal_deadline_passed
  2004  duplicate_commit
  2005  commitment_mismatch
  2006  price_above_max

3xxx  执行错误
  3001  no_capacity
  3002  account_banned
  3003  upstream_rate_limited
  3004  upstream_error

4xxx  结算错误
  4001  htlc_not_funded
  4002  insufficient_balance
  4003  ticket_signature_invalid
  4004  ticket_sequence_violation

5xxx  系统错误
  5001  internal_error
  5002  storage_error
```

---

## 8. 测试向量

每个实现必须能通过 `tests/vectors/` 目录下的测试向量：

```
tests/vectors/
├── signatures/
│   ├── listing_sign_001.json
│   └── listing_sign_001_expected_sig.bin
├── commitments/
│   ├── commit_001_input.json
│   └── commit_001_expected_hash.hex
├── reveal_encryption/
│   ├── reveal_001_inputs.json
│   └── reveal_001_expected_ciphertext.bin
└── tickets/
    └── ...
```

> 这些向量由 reference Rust 实现生成，Go 实现必须能 byte-for-byte 复现。

---

## 9. 版本协商

- 客户端读到协议消息时检查 `pv` 字段
- `major.minor` 必须匹配；`patch` 不同允许通信
- 不匹配返回 `1005 protocol_version_mismatch`
- 双方都应在 UI / 日志中显示对方的 pv

---

## 10. 修改流程

任何对本文档的修改：

1. 开 PR，标题 `proto: <一句话描述>`
2. 必须附带：
   - 改动理由
   - 影响的 Rust 文件清单
   - 影响的 Go 文件清单
   - 测试向量是否需要更新
3. Reviewer + 至少一个第三方 reviewer 通过才能 merge
4. Merge 后立即写 changelog
