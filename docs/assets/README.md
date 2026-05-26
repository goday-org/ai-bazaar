# docs/assets

README 与各文档引用的截图、动图、视频素材放在这里。

## 命名约定

- `demo-cli.gif` — buyer-cli 跑一次完整成交的录屏（W7 后补）
- `demo-gui.gif` — buyer-client Tauri GUI 主流程（W8 后补）
- `seller-ctl.gif` — 卖家挂单 / 抢单流程（W9 后补）
- `architecture-overview.png` — 架构总览（如不用 Mermaid 时的兜底图）
- `vickrey-example.png` — Vickrey 二价机制示例图

## 录屏建议

- 终端动图：用 [vhs](https://github.com/charmbracelet/vhs) 生成 `.gif`，
  保证可重现、字号统一、时长 ≤ 30s。
- GUI 截图：1280×800 retina，PNG 压缩用 `pngquant`。
- 视频：MP4 H.264，≤ 10MB。GitHub README 里直接拖拽到 issue/PR 评论框
  生成 `https://github.com/.../assets/...` 链接即可，**不用提交进 git**。

## 不要提交进仓库的内容

- 含真实 API key / OAuth token 的录屏帧
- 任何 `sk-`、`GOCSPX-`、`ghp_`、助记词
- 涉及具体上游账号 ID 的截图
