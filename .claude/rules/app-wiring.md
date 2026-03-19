---
paths:
  - "app/**"
---

# App Wiring Rules

- Background jobs MUST be guarded with a condition (feature flag or config check) before starting.
- Configuration MUST use `configutil.NewPrefixedConfigProvider` + `configutil.NewPrefixedConfigInfoProvider` — no raw env reads or ad-hoc config parsing.
- Prefer `jobber` for regular periodic/loop jobs — it handles lifecycle automatically.
- No graceful shutdown via context cancellation. Custom jobs MUST follow this pattern:
  - `Run() error` and `Close() error` methods. Both return errors on invalid state transitions or internal errors.
  - `atomic.Int32` state field with states: idle(0) → starting(1) → running(2) → stopping(3). Prevents multiple concurrent runs.
  - `Run` transitions idle → starting, creates a `context.WithCancel` and `quitCh`, then transitions to running.
  - `Close` transitions running → stopping, calls `cancel()` to interrupt in-flight work, then writes to `quitCh` (blocking send, not close) to wait until `Run` exits. `Run` resets state to idle in defer.
  - All long operations inside the job must accept `ctx` so `cancel()` can interrupt them.
  - Job must be reusable — channels and context created fresh in `Run`, not in constructor.
