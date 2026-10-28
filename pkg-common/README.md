# pkg-common

Shared Go building blocks for Fusewall services in this repo: PASETO tokens, a structured logger, and gin middleware (auth, panic recovery, rate limiting).

Rules:
- **No product names.** Cookie names, issuers and limits are passed in as config.
- **Only add.** Other apps in this repo build against this folder, so never change what an existing function does; add a new one instead.
- Its tests run in CI for every app that uses it.

It started as a trimmed copy of Showtime's `pkg-common` (without the Redis, queue, tailscale and message-pipe code). The two copies are independent; see `docs/DECISIONS.md`.
