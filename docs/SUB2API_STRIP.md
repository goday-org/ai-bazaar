# sub2api 剥离指南（SUB2API_STRIP.md）

> 把 Wei-Shaw/sub2api fork 成最小的 `seller-relay`，砍掉 80% 文件但保留所有反封号经验。

---

## 0. 阅读前提

- 你必须已读 ARCHITECTURE.md 和 PITFALLS.md
- 你了解 sub2api 是 Go + Gin + Ent + PostgreSQL + Redis
- 你了解 Google Wire 和 Ent ORM 的代码生成机制

---

## 1. License 与归属

- sub2api 协议：**LGPL-3.0**
- 你 fork 后必须：
  - 保留原 LICENSE 文件，不要删
  - 在 `seller-relay/README.md` 顶部写明 "Forked from [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) (LGPL-3.0)"
  - 修改文件时保留原文件头版权（如果有）
- AI Bazaar 整体仓库用 **AGPL-3.0**，与 LGPL-3.0 兼容（AGPL 严于 LGPL）

---

## 2. 基线冻结

### 2.1 第一步：纯 clone，不改任何代码

```bash
mkdir -p seller-relay
cd seller-relay
git clone --depth=1 https://github.com/Wei-Shaw/sub2api.git tmp-upstream
mv tmp-upstream/backend/* .
mv tmp-upstream/backend/.dockerignore tmp-upstream/backend/.golangci.yml . 2>/dev/null || true
mv tmp-upstream/LICENSE LICENSE
rm -rf tmp-upstream
cd ..
git add seller-relay
git commit -m "chore(strip): import sub2api backend baseline (LGPL-3.0)"
```

### 2.2 验证基线可编译

```bash
cd seller-relay
rtk go mod download
rtk go build ./...
rtk go vet ./...
```

**如果上面任何一步失败，先修这些问题再继续。不要带着编译错误开始剥离。**

### 2.3 跑现有测试

```bash
rtk go test ./internal/integration/... -run TestGatewayE2E
```

记录通过的测试列表，剥离过程中要保持这些通过。

---

## 3. 九步剥离（每步一个 commit，必须按顺序）

每完成一步都要：
- `rtk go build ./...` 通过
- `rtk go vet ./...` 通过
- `rtk go test ./...` 不引入新失败（已有失败可记录在 KNOWN_FAILURES.md）
- 单独 commit，commit message 用 `strip(step-N): <描述>`

### Step 1 — 删除路由

**删除文件**：
- `internal/server/routes/admin.go`
- `internal/server/routes/auth.go`
- `internal/server/routes/user.go`
- `internal/server/routes/payment.go`
- `internal/server/routes/redeem.go`（若存在）

**修改文件**：
- `internal/server/router.go`：删掉对应的 `setupAdminRoutes / setupAuthRoutes / setupUserRoutes / setupPaymentRoutes` 调用

**验收**：
```bash
rtk go build ./... 2>&1 | grep -c "undefined" # 应 > 0，因为 handler 还在但路由没了
# 这步会有编译错误，正常，下一步修复
```

### Step 2 — 删除 admin/payment/user handler

**删除目录 / 文件**：
- `internal/handler/admin/`
- `internal/handler/auth_*.go`
- `internal/handler/payment_*.go`
- `internal/handler/redeem_*.go`
- `internal/handler/subscription_*.go`
- `internal/handler/announcement_*.go`
- `internal/handler/setting_*.go`
- `internal/handler/user_handler.go`
- `internal/handler/totp_handler.go`
- `internal/handler/api_key_handler.go`（API key 管理 UI，保留 service 层）
- `internal/handler/available_channel_handler.go`
- `internal/handler/channel_monitor_user_handler.go`
- `internal/handler/page_handler.go`
- `internal/handler/usage_*.go`（保留 service 层的 usage logging，删掉对外查询）

**修改 `internal/handler/wire.go`**：删掉上述文件提供的 provider

**验收**：
```bash
rtk go build ./...  # 仍会失败（service 层未删），但 handler 编译应通过
```

### Step 3 — 删除 service 层的支付 / 订阅 / 推广

**删除文件**（在 `internal/service/`）：

- `payment_*.go`
- `subscription_*.go`
- `user_subscription*.go`
- `redeem_*.go`
- `promo_*.go`
- `affiliate_*.go`
- `balance_notify_*.go`
- `notification_email_*.go`
- `email_*.go`

**修改 `internal/service/wire.go`**：删掉上述 provider

**验收**：`rtk go build ./...` 仍可能失败（依赖未清），但应少 200+ undefined

### Step 4 — 删除 service 层的用户 / 认证 / admin

**删除文件**：

- `internal/service/user.go`
- `internal/service/user_service*.go`
- `internal/service/user_attribute*.go`
- `internal/service/user_group_rate*.go`
- `internal/service/auth_*.go`
- `internal/service/identity_*.go`
- `internal/service/admin_*.go`
- `internal/service/totp_service.go`
- `internal/service/turnstile_service.go`
- `internal/service/registration_email_policy*.go`
- `internal/service/content_moderation*.go`
- `internal/service/announcement_*.go`
- `internal/service/group_capacity_service.go`
- `internal/service/data_management_*.go`
- `internal/service/backup_*.go`
- `internal/service/update_*.go`
- `internal/service/system_operation_lock_*.go`

**修改**：
- `internal/service/api_key_service.go` 简化：去掉用户关联，API key 直接对应一个"endpoint"概念
- `internal/server/middleware/api_key_auth.go` 简化：只校验 key 存在 + 未过期，不查 user

### Step 5 — 删除 service 层的运营 / 仪表盘

**删除文件**：

- `internal/service/ops_*.go`（全部，约 30 个文件）
- `internal/service/dashboard_*.go`
- `internal/service/pricing_*.go`（**保留 `model_pricing_resolver.go` 用于计量**）
- `internal/service/billing_*.go`（**保留 `gateway_billing_*.go`**）
- `internal/service/parse_integral_number_unit.go`

### Step 6 — 删除 service 层的渠道质量监测

**删除文件**：

- `internal/service/channel_monitor_*.go`
- `internal/service/scheduled_test_*.go`
- `internal/service/account_test_service*.go`
- `internal/service/crs_sync_*.go`
- `internal/service/proxy_probe_*.go`

**保留**：
- `internal/service/proxy_service.go`（被 transport 调用）
- `internal/service/tls_fingerprint_profile_service.go`（**这个绝对不能删**，反爬关键）
- `internal/service/proxy_latency_cache.go`（如果被 transport 引用）

### Step 7 — 重跑 Wire

```bash
cd seller-relay
rtk go install github.com/google/wire/cmd/wire@latest
cd internal/service
wire
cd ../handler
wire
cd ../../cmd/server
wire
```

**如果 wire 报错**：
- 多半是某个保留的 service 依赖了你删掉的 service
- 解决方案：
  1. 优先把那个保留 service 的依赖改为接口 + noop 实现
  2. 实在不行，把它一起删（确认它不在 gateway hot path 上）

**绝对不能为了 wire 通过把支付 / 用户 service 加回来。**

### Step 8 — Schema 剪枝 + 重跑 Ent

**删除 schema 文件**（在 `ent/schema/`）：

- `announcement*.go`
- `payment_*.go`
- `promo_code*.go`
- `redeem_code.go`
- `subscription_plan.go`
- `user_subscription.go`
- `user_attribute_*.go`
- `auth_identity*.go`
- `identity_adoption_decision.go`
- `pending_auth_session.go`
- `channel_monitor*.go`
- `usage_cleanup_task.go`
- `payment_audit_log.go`

**保留 schema**：
- `account.go`、`account_group.go`、`api_key.go`、`group.go`、`setting.go`
- `proxy.go`、`security_secret.go`、`tls_fingerprint_profile.go`
- `error_passthrough_rule.go`、`idempotency_record.go`
- `usage_log.go`（保留计量记录）
- `mixins/`（保留所有 mixin）

**重跑 ent generate**：

```bash
cd seller-relay
rtk go generate ./ent/...
```

**如果生成失败**：检查是否还有保留的 schema 引用了删掉的 schema（外键）。需要去掉这些外键。

### Step 9 — 数据库降级到 SQLite

**目标**：完全去掉 PostgreSQL + Redis 依赖。

#### 9.1 替换 PostgreSQL → SQLite

**修改 `internal/config/config.go`**：
- 删除 `Database.Driver` 选项，写死 `sqlite`
- `Database.DSN` 默认 `~/.ai-bazaar/seller.db`

**修改 `internal/repository/ent.go` 或 `db_pool.go`**：

```go
// 旧：
import _ "github.com/lib/pq"
client, err := ent.Open("postgres", cfg.Database.DSN)

// 新：
import _ "modernc.org/sqlite"
client, err := ent.Open("sqlite3", cfg.Database.DSN+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
```

**Ent 配置**：检查 `ent/generate.go` 是否需要改 dialect：
```go
//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ./schema
```

**Postgres-only 特性的降级**：
- JSONB 索引 → 普通列 + 应用层过滤
- 部分唯一索引（partial unique）→ 普通唯一索引（如果业务允许）或应用层校验
- Array 类型 → JSON 字符串

如果遇到强依赖 PostgreSQL 的 service（如某个用 GIN 索引的全文搜索），**降级为内存实现或直接删该 service**。

#### 9.2 删除 Redis 依赖

**Redis 在 sub2api 里主要用于**：
- API key 缓存
- 账户调度状态
- RPM / 并发计数

**降级方案**：

| Redis 用途 | 替换为 |
|-----------|-------|
| `api_key_auth_cache` | 进程内 `sync.Map` + TTL |
| `scheduler_cache` / `scheduler_outbox` | 直接读 SQLite |
| `rpm_cache` / `user_rpm_cache` | `golang.org/x/time/rate` 内存版 |
| `session_limit_cache` | `sync.Map` |
| `*_counter_cache` | atomic + map |
| `gemini_token_cache` / `refresh_token_cache` | SQLite + 进程内 cache |

**实现策略**：
- 定义一个 `Cache` interface
- 提供两个实现：`MemoryCache`（默认）和 `RedisCache`（保留代码但不默认启用）
- 通过 wire 注入选择

**修改 `internal/config/config.go`**：
- `Redis.Enabled` 默认 `false`
- 启动时如果 `Redis.Enabled=false`，跳过 Redis 连接

#### 9.3 Migration 重建（**不是手工 cat sql**）

sub2api 的 migration 由 ent 自动生成（不是手写 SQL）。直接 `cat` 拼接源 SQL 文件不可行——schema 已经被裁剪，旧 migration 引用了已删除的表。

**正确做法**：

```bash
cd seller-relay

# 1. 备份现有 migrations 目录（仅供查阅）
mv migrations migrations.legacy

# 2. 起一个干净的 SQLite 数据库做 baseline
mkdir -p migrations
sqlite3 /tmp/seller-relay-baseline.db ".databases"

# 3. 用 ent 在裁剪后的 schema 基础上生成 init migration
go run -mod=mod entgo.io/ent/cmd/ent new --target ./ent/schema init  # 仅在缺时
go run -mod=mod ariga.io/atlas/cmd/atlas migrate diff init \
    --dir "file://migrations?format=atlas" \
    --to "ent://ent/schema" \
    --dev-url "sqlite:///tmp/seller-relay-baseline.db?_fk=1"

# 4. 验证生成的 migration 能干净 apply
go run -mod=mod ariga.io/atlas/cmd/atlas migrate apply \
    --dir "file://migrations?format=atlas" \
    --url "sqlite://${HOME}/.ai-bazaar/seller.db?_fk=1"
```

**结果**：
- `migrations/` 下生成 1 个 `*_init.sql` + `atlas.sum`
- `migrations.legacy/` 留作参考（commit 时 .gitignore 它）

**Atlas 是 ent 官方推荐的 migration 工具**，比手写 SQL 安全（自动 checksum / 拒绝偏差）。

**验收**：
```bash
rm -f ~/.ai-bazaar/seller.db
go run ./cmd/server  # 启动应成功，自动 apply migration 建表
sqlite3 ~/.ai-bazaar/seller.db ".tables"  # 应列出保留的所有表
```

---

## 4. 加入 IPC（Control Plane）

完成九步剥离后，加上 seller-ctl 通信入口。

### 4.1 新增包：`internal/ipc/`

```go
// internal/ipc/server.go
package ipc

import (
    "encoding/json"
    "net"
    "os"
    "path/filepath"
)

func StartServer(socketPath string, handler RPCHandler) error {
    os.MkdirAll(filepath.Dir(socketPath), 0700)
    os.Remove(socketPath)
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        return err
    }
    os.Chmod(socketPath, 0600)
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        go handleConn(conn, handler)
    }
}
```

### 4.2 注册 4 个 RPC

按 `PROTOCOL.md §6.3` 实现：

- `OpenEndpoint`
- `CloseEndpoint`
- `GetUsage`
- `ListAccounts`

**实现要点**：
- `OpenEndpoint`：
  1. 调用 `api_key_service.Create()` 生成一个临时 API key
  2. 关联 buyer_pubkey、service_id、quota_tokens
  3. 在 SQLite 写入 endpoint 记录
  4. 返回 `local_url = "http://127.0.0.1:" + port`
  5. 调度 gateway 监听该 API key
- `CloseEndpoint`：撤销 API key，gateway 拒绝该 key 的后续请求
- `GetUsage`：从 usage_log 表读出累计 token 数
- `ListAccounts`：查 account 表

### 4.3 启动入口

修改 `cmd/server/main.go`：

```go
// 启动 HTTP gateway（保留）
go startHTTPServer(cfg)
// 启动 IPC（新增）
ipcSocket := filepath.Join(os.Getenv("HOME"), ".ai-bazaar", "seller.sock")
go ipc.StartServer(ipcSocket, &ipc.Handler{
    AccountService: accountSvc,
    APIKeyService:  apiKeySvc,
    UsageService:   usageSvc,
})
```

---

## 5. 必须保留的"硬常量"

剥离过程中**绝对不能删**的常量与逻辑（每条都来自 PITFALLS.md）：

### Claude / Anthropic

- `internal/pkg/claude/constants.go` 整个文件 → 不动
- `CLICurrentVersion` 当前 `"2.1.92"` → 跟随 upstream
- User-Agent / anthropic-beta / X-Stainless-* 完整 header 集
- 模型 ID 映射表（短名 → 长 ID）
- Haiku 特例分支

### OpenAI

- `internal/pkg/openai/oauth.go` 全部常量
- `ClientID = "app_EMoamEEZ73f0CkXaXp7hrann"`
- Refresh scope **不带 offline_access**
- `instructions.txt` 模板原样

### Gemini / Antigravity

- `internal/pkg/antigravity/oauth.go` 全部常量
- `internal/pkg/geminicli/` 全部
- 双 base URL（prod / daily sandbox）+ 5min 可用性冷却

### 反爬虫

- `internal/pkg/tlsfingerprint/` 全部
- `internal/service/tls_fingerprint_profile_service.go`

### 计费一致性

- Sticky session 切账户时的 `ForceCacheBilling` 逻辑
- 四个错误计数器（401/403/429/internal500）
- UMQ（User Message Queue）

---

## 6. 验收

剥离完成后，跑这套验收：

```bash
# 1. 编译通过
rtk go build ./...

# 2. 静态检查
rtk go vet ./...
rtk go install honnef.co/go/tools/cmd/staticcheck@latest
rtk staticcheck ./...

# 3. 单元测试
rtk go test ./internal/pkg/...
rtk go test ./internal/service/... -run "TestSticky|TestUMQ|TestAccount|TestRate"

# 4. 文件统计
find . -name '*.go' ! -name '*_test.go' | wc -l   # 期望 < 280
find . -name '*.go' ! -name '*_test.go' | xargs wc -l | tail -1  # 总行数期望 < 50000

# 5. 启动冒烟
rtk go run ./cmd/server &
SERVER_PID=$!
sleep 2
curl -s http://127.0.0.1:8000/health  # 应返回 200
kill $SERVER_PID

# 6. IPC 冒烟
echo '{"jsonrpc":"2.0","id":1,"method":"ListAccounts","params":{}}' | nc -U ~/.ai-bazaar/seller.sock
# 应返回 {"jsonrpc":"2.0","id":1,"result":{"accounts":[]}}
```

---

## 7. 完成后的 STOP

走完这套剥离 + IPC，**停下来**等 reviewer 看：

1. 提交 milestone 总结（`docs/progress/W1-W2.md`）
2. 列出删除文件数 / 保留文件数 / 总行数
3. 跑一遍 §6 验收清单贴输出
4. 列出 KNOWN_FAILURES.md 里的所有遗留问题
5. **明确写**：哪些 sub2api 上游变更将来需要合并

等 `APPROVED 进入 W3` 后再继续。

---

## 8. 与上游 sub2api 同步策略

**为什么要保留同步能力**：Anthropic / OpenAI 协议每月变，sub2api 维护者会先解决。

### 8.1 设置 upstream remote

```bash
cd seller-relay
rtk git remote add upstream https://github.com/Wei-Shaw/sub2api.git
```

### 8.2 同步流程

未来想拉 upstream 更新时：

```bash
rtk git fetch upstream main
rtk git checkout -b sync-upstream-YYYYMMDD
rtk git cherry-pick <commit-hashes>  # 仅挑反爬 / 协议常量相关
# 解决冲突
rtk go build ./...
rtk go vet ./...
# 通过则 PR 到 main
```

**只 cherry-pick 这些路径**：
- `internal/pkg/claude/`
- `internal/pkg/openai/`
- `internal/pkg/gemini/`
- `internal/pkg/geminicli/`
- `internal/pkg/antigravity/`
- `internal/pkg/apicompat/`
- `internal/pkg/tlsfingerprint/`

**不 cherry-pick** 用户 / 支付 / admin 相关变更。

---

## 9. 常见问题

**Q：删某个 service 后 wire 报循环依赖怎么办？**
A：先把那个 service 改成接口，给一个 noop 实现注入。等所有依赖清完再考虑是否真的删除。

**Q：Ent generate 提示 schema 字段不兼容怎么办？**
A：检查是否有保留的 schema 引用了删掉 schema 的外键 / Edge。把这些 Edge 去掉。

**Q：sqlite WAL mode 在 macOS 上偶发锁死？**
A：把 `busy_timeout` 调到 30 秒（30000）。极端情况下用 BEGIN IMMEDIATE 替代 BEGIN。

**Q：发现某个上游 API 请求体的字段在 sub2api 里被 sub2api 自己改过（不是简单透传）？**
A：保留 sub2api 的改写，并加测试覆盖。这种改写通常是为了过上游风控。
