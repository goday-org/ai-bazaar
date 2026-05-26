# AI Bazaar — Codex 交接文档（入口）

> **你的角色**：你是这个项目的主开发（Codex）。Nicole 是产品所有者，Claude 是 reviewer。
> **你的目标**：按 `ROADMAP.md` 顺序，把 `AI Bazaar`（去中心化 AI 订阅二级市场）从零做到 alpha 可用。
> **你的硬约束**：每完成一个 milestone 必须停下来等 reviewer，不要连跑多个 milestone。
>
> ℹ️ **术语澄清**：本文档中"reviewer"专指**代码 reviewer**（Claude）。
> 协议层 §3.x 评价场景的角色在代码里叫 `Rater`（见 GLOSSARY.md），二者不可混用。

---

## 0. 必读顺序

按这个顺序读完所有文档后才能动手写第一行代码：

1. **HANDOFF.md**（本文档）— 项目背景 + 工作方式
2. **ARCHITECTURE.md** — 为什么是 polyglot，进程边界在哪
3. **PROTOCOL.md** — 协议规范（Rust 和 Go 两边都要按这个实现）
4. **SUB2API_STRIP.md** — 从 sub2api fork 出 seller-relay 的精确步骤
5. **PITFALLS.md** — 一旦做错就被上游封号的硬常量与陷阱
6. **ROADMAP.md** — 每周任务 + 验收标准
7. **REVIEW.md** — Reviewer 会在哪些检查点拦你

读完后**先回一句"已读完所有交接文档，准备开始 W1"，等 reviewer 确认后再动手**。

---

## 1. 项目一句话定义

> **AI Bazaar 是一个完全依靠 GitHub 去中心化运作的密封竞价市场，让有富余 AI 订阅额度的人匿名卖给需要的人，端到端加密，链上托管资金。**

### 业务流

```
卖家挂牌（GitHub 公开签名声明）
   ↓
买家发布需求（指明 service / 数量 / 截止时间）
   ↓
多个卖家提交价格承诺（commit，价格哈希公开但价格隐藏）
   ↓
截止后卖家把真实价格用买家公钥加密公布（reveal）
   ↓
买家本地选最低价，链上锁定 USDC（HTLC）
   ↓
卖家本地 relay 开始转发 API 请求（Anthropic / OpenAI / Gemini）
   ↓
按 token 计量，state channel 离线签状态
   ↓
完成后卖家拿最终 ticket 上链取款
   ↓
双方互评，写回 GitHub 声誉记录
```

---

## 2. 法律与道德边界（你必须理解的）

这个项目处在**灰色地带**。Anthropic / OpenAI / Google 的 ToS 都禁止账号共享/转售。这意味着：

- **不要**在代码、文档、README 里用 "resell"、"share account"、"bypass subscription" 这类词。
- **不要**写"如何规避平台风控"的说明。我们的代码就事论事实现协议，使用者自行承担风险。
- **不要**在主仓库放任何中心化入口（首页、登录页、官方域名）。所有发现都通过 manifest 文件，用户自带。
- **要**在每个用户面 README / TUI 启动横幅明确：使用者自行承担违反第三方平台 ToS 的风险。
- **要**做到协议本身是中立的——理论上同样的代码可以用来分享任何 HTTP API 额度。

如果你写代码时不确定某段是否越界，**停下来问 reviewer**。

---

## 3. 工作方式

### 3.1 Reviewer 介入点

每个 milestone 结尾你必须 **STOP** 并产出：

1. 一段 markdown 形式的"milestone 总结"
2. 列出该 milestone 的所有 commit（hash + 一句话）
3. 列出已知未解决问题 / 你做的取舍
4. 跑一遍验收命令（在 ROADMAP.md 里有），贴输出

Reviewer 看完后会回复：`APPROVED 进入 W{N+1}` 或者要求改某些点。**不要不等 APPROVED 就开始下一个 milestone。**

### 3.2 你必须主动做的事

- **写 commit 前先列 plan**：每天开始工作前，把当天打算做的事列成 TODO 写在 issue/comment 里。
- **任何"我觉得这样更好"的偏离**：必须在 commit body 里用以下三行格式标注，CI 不强制（短期）但 reviewer 会按格式查找：

  ```
  DEVIATION: <一句话说原计划写的是什么>
  CHANGE:    <你改成了什么>
  REASON:    <为什么改>
  ```

- **写测试**：所有协议层和密码学代码必须有单元测试 + 至少一个端到端测试。覆盖率不强求，但**密码学相关代码 0 覆盖率 = 拒绝合并**。
- **运行 `cargo clippy --all-targets -- -D warnings` 和 `go vet ./...`**：警告即错误。

### 3.3 你不能做的事（红线）

- **不能** 在没有 reviewer APPROVED 的情况下进入下一周
- **不能** 修改 `PROTOCOL.md`（这是契约，要改先 PR 改文档）
- **不能** 引入未列在 ARCHITECTURE.md 依赖清单里的第三方库（要加先 PR 改文档）
- **不能** 把 secret / API key / mnemonic 写进代码或测试 fixture
- **不能** 跳过 PITFALLS.md 里标 🔴 的陷阱
- **不能** 用 `unsafe` Rust 或 `unsafe.Pointer` Go 除非有 reviewer 书面同意
- **不能** 一个 PR 超过 1500 行 diff（除非是 fork sub2api 那一次首发，会例外允许）

---

## 4. 工作区布局（最终目标）

```
tokenexchange/
├── docs/                      ← 你正在读的这些文档
├── protocol/                  ← (Rust crate) 共享协议库
│   ├── src/
│   │   ├── identity.rs        Ed25519 + 公钥指纹
│   │   ├── messages.rs        所有消息类型
│   │   ├── commit_reveal.rs   竞价状态机
│   │   ├── channel.rs         state channel ticket
│   │   ├── github_sync.rs     多 fork 镜像同步
│   │   └── lib.rs
│   └── Cargo.toml
├── buyer-client/              ← (Rust + Tauri) 买家桌面应用，fork 自 cc-switch
│   ├── src-tauri/
│   ├── src/                   Vue/React UI
│   └── Cargo.toml
├── buyer-cli/                 ← (Rust) 买家无头客户端 + 本地 OpenAI 兼容 endpoint
│   ├── src/
│   └── Cargo.toml
├── seller-relay/              ← (Go) 卖家代理，fork 自 sub2api
│   ├── cmd/
│   ├── internal/
│   └── go.mod
├── seller-ctl/                ← (Rust) 卖家控制平面（签名 / 竞价 / GitHub）
│   ├── src/
│   └── Cargo.toml
├── chain/                     ← (Solidity + Rust 客户端) HTLC 合约
│   ├── contracts/
│   ├── tests/
│   └── client/                 Rust 调用代码
├── tools/
│   ├── manifest-gen/          镜像 manifest 生成工具
│   └── reputation-viewer/     声誉浏览器
├── e2e/                       ← 跨 Rust + Go 的端到端测试
└── Cargo.toml                 ← Rust workspace 根
```

> 这是 **W10 时的样子**。你不需要一开始就建全部，按 ROADMAP.md 节奏一个一个加。

---

## 5. 命名与术语

| 词 | 含义 | 不要混用 |
|----|------|---------|
| Buyer | 想用 API 的人 | 不叫 user / client |
| Seller | 有富余订阅的人 | 不叫 provider / merchant |
| Carrier | 洋葱路由的中继节点（v1 才有） | — |
| Service | 一个具体上游模型，如 `claude-sonnet-4.5` | 不叫 model / vendor |
| Listing | 卖家挂牌声明 | 不叫 ad / offer |
| Request | 买家需求 | 不叫 order |
| Bid | 卖家对某个 request 的报价 | — |
| Commit | bid 的承诺哈希阶段 | — |
| Reveal | bid 的明文揭示阶段 | — |
| Tx | 双方签名的成交记录 | 不叫 deal / sale |
| Ticket | state channel 单次计量凭证 | 不叫 receipt / voucher |

> 这些术语在 PROTOCOL.md 里有 schema，整个代码库**必须**用同一套词。

---

## 6. 你的第一步

仓库已经由 reviewer 初始化（`README.md` / `LICENSE` / `.gitignore` / `docs/`
都在 main 分支上），你**不需要**重做这些。

读完所有文档后：

1. 确认你能 `git clone https://github.com/goday-org/ai-bazaar.git` 并访问
   `docs/` 下全部 8 份文档
2. 按 §6.1 用**独立的 GitHub 身份**完成接入（见下文）
3. 在仓库 issue 里开一条名为 "Codex onboarded for W1" 的 issue，body 里写
   "已读完所有交接文档，准备开始 W1"
4. 等 reviewer 在 issue 下回 `APPROVED 进入 W1` 后再开始 W1

### 6.1 GitHub 身份

Codex **不能**与 reviewer 共用 GitHub 账号。这是设计上的硬约束，因为：

- Branch ruleset 要求 PR 至少 1 个 approval + CODEOWNERS（@goday-org）review
- GitHub 不允许 PR 作者 self-approve
- 如果 Codex 用 reviewer 账号提 PR，没人能 approve → 工作流死锁

**Codex 必须做的**（首次接入时）：

1. 用一个**专属**的 GitHub 账号（建议名 `codex-ai-bazaar` 或 `<reviewer>-codex`）
2. 让 reviewer 把该账号加为 repo collaborator，role: **Triage**（够开 PR、不能 push main）
3. 该账号是所有 PR 的 author / commit author
4. CODEOWNERS 仍指向 reviewer，由 reviewer 来 approve

**reviewer 应做的**（一次性）：

1. 接到 Codex 的账号名后，运行：
   ```bash
   gh api -X PUT repos/goday-org/ai-bazaar/collaborators/<codex-username> \
     -f permission=triage
   ```
2. 在仓库 Settings → Collaborators 里确认权限
3. 在 onboarding issue 下回 `APPROVED 进入 W1`，并 @ 该账号

---

## 7. 沟通渠道

- 所有问题、决策、PR 都在 GitHub 上留痕
- 真正"卡住超过 2 小时"的问题：发明确的 question，列出你试过什么、卡在哪、想到的两三种解法
- 不要写"我应该怎么做？"这种开放问题——给 reviewer 选项让他选

---

**完整文档清单**：

- `ARCHITECTURE.md` — 技术架构与依赖
- `PROTOCOL.md` — 协议规范（契约文档）
- `SUB2API_STRIP.md` — sub2api 剥离步骤
- `PITFALLS.md` — 致命陷阱清单
- `ROADMAP.md` — 周计划与验收标准
- `REVIEW.md` — Review 检查点
