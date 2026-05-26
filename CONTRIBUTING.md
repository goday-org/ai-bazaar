# Contributing to AI Bazaar

> Pre-alpha. The implementing agent is **Codex**; the reviewer is **Claude**;
> the product owner is **Nicole**. External contributions are not yet open.

---

## Workflow

All implementation work follows [`docs/ROADMAP.md`](docs/ROADMAP.md) and is
gated by [`docs/REVIEW.md`](docs/REVIEW.md). Read [`docs/HANDOFF.md`](docs/HANDOFF.md)
first if you have not already.

## Branching

- `main` is protected. No direct pushes; everything goes through PR.
- Feature branches: `w{N}/<short-topic>` for milestone work
  (e.g. `w1/strip-routes`, `w4/identity-keypair`).
- Fix-up branches: `fix/<topic>` for bug fixes between milestones.

## Commit messages

Conventional Commits, enforced by CI:

```
<type>(<scope>): <subject under 72 chars>

<optional body explaining the why>

<optional footer e.g. DEVIATION / Refs / Co-authored-by>
```

Allowed types: `chore`, `docs`, `feat`, `fix`, `refactor`, `test`, `build`,
`ci`, `perf`, `revert`, `strip`, `proto`.

`strip` is reserved for the sub2api strip-down commits; `proto` for changes
to the protocol contract under `docs/PROTOCOL.md`.

If your commit deviates from the plan in `ROADMAP.md` or contradicts
guidance in `PITFALLS.md`, the body **must** start with:

```
DEVIATION: <plan said X, you did Y>
REASON: <why>
```

## PR rules

- One PR per logical change. Hard cap: 1500 lines added (CI enforces).
  The W1-W2 sub2api baseline import is the only allowed exception, and
  the PR title must contain `strip`, `sub2api import`, or `baseline`.
- Every PR auto-requests review from CODEOWNERS.
- A PR may only be merged after `APPROVED` from the reviewer.
- Squash-merge by default; preserve the conventional commit subject.

## Bilingual README rule

`README.md` (English) and `README.zh-CN.md` (Simplified Chinese) are the
project's public landing pages and must stay in content parity.

- **Any change to one MUST be applied to the other in the same PR.**
  This includes: badges, sections, diagrams, tables, milestone status,
  doc links, disclaimers.
- Mermaid diagrams are language-agnostic — copy verbatim, only translate
  inline labels if they contain prose.
- The reciprocal language switcher at the top of each file must be kept
  intact:
  - `README.md` opens with: `**Language**: **English** · [简体中文](README.zh-CN.md)`
  - `README.zh-CN.md` opens with: `**Language**: [English](README.md) · **简体中文**`
- If you only have time / capacity for one language, **don't touch either**.
  Open an issue describing the desired change instead.

Reviewer rejects PRs that drift the two files apart.

## CI checks

The following must be green before merge:

- `docs / link check` — broken internal links fail.
- `docs / markdown lint` — basic style.
- `docs / forbidden phrasing` — legal-risk word scan.
- `docs / protocol version present` — contract version is declared.
- `secrets / gitleaks` — secret scanner with project rules in
  `.github/gitleaks.toml`.
- `commit-conventions` — every non-merge commit subject matches the type
  whitelist.
- `commit-conventions / PR size guard` — 1500-line cap.

Once code lands (`W3+`), this list will grow with `cargo test`, `go test`,
`cargo clippy`, and `gosec` jobs.

## Local checks before opening a PR

```bash
# Markdown links (offline)
npx -y lychee --offline --include-fragments docs/**/*.md README.md

# Markdown lint
npx -y markdownlint-cli2 'docs/**/*.md' README.md

# Secrets
gitleaks detect --config .github/gitleaks.toml --no-banner

# Conventional commits on your branch
git log --pretty='%s' main..HEAD
```

## Reporting security issues

Do not open a public issue. Email the product owner directly (contact
listed in profile of `@goday-org`).

## License

By contributing, you agree your contributions are licensed under
[AGPL-3.0](LICENSE).
