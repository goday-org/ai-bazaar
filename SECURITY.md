# Security Policy

## Supported versions

This project is in **pre-alpha**. No version has been audited or released
for production use. Do not rely on it for handling real funds or sensitive
data.

## Reporting a vulnerability

If you find a security issue, **do not open a public issue**. Instead:

1. Email the maintainer: contact listed on the GitHub profile of
   [@goday-org](https://github.com/goday-org).
2. Include:
   - A description of the vulnerability and impact.
   - Steps to reproduce (or a minimal proof-of-concept).
   - Your suggested fix, if any.
3. Allow 90 days for a response before public disclosure.

We will acknowledge receipt within 7 days and aim to provide a remediation
plan within 30 days for confirmed issues.

## Out of scope

The following are **known limitations**, not vulnerabilities:

- A seller running the relay can read plaintext API requests proxied
  through it. End-to-end encryption against the seller is not a goal of
  v0 (see [`docs/PITFALLS.md` §8](docs/PITFALLS.md)).
- Linkability of buyer / seller fingerprints across requests until
  carrier-based onion routing ships in v1.
- The Antigravity OAuth `client_secret` is publicly documented in the
  upstream sub2api repository. It can be revoked by the provider at any
  time. Use `ANTIGRAVITY_OAUTH_CLIENT_SECRET` to override at runtime
  (see [`docs/PITFALLS.md` §1.3](docs/PITFALLS.md)).
- Anything that depends on third-party platform terms of service (this
  project does not warrant any specific upstream platform behavior).

## Cryptographic primitives

| Use | Algorithm |
|-----|-----------|
| Identity signing | Ed25519 |
| Key exchange | X25519 |
| Symmetric AEAD | ChaCha20-Poly1305 |
| Hash | BLAKE3, SHA-256, SHA-512 (Ed25519 internal) |
| KDF | HKDF-SHA256 |
| Seed mnemonic | BIP-39 (24 words) |

Implementations must use audited crates (see
[`docs/ARCHITECTURE.md` ADR-006](docs/ARCHITECTURE.md)). Hand-rolled
cryptography is rejected at review.

## Known wallet/fund safety boundaries

- Single-tx cap during alpha: **$50**.
- HTLC timelock: 1–24 hours depending on amount.
- USDC custody is subject to Circle's blacklist policy; the refund path
  does not depend on seller cooperation.

See [`docs/PITFALLS.md` §5](docs/PITFALLS.md) for the full fund-safety
checklist.
