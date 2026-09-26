# CALLBACK BEHAVIOR AUDIT

| Callback Pattern | Python behavior | Go handler | Tested | Status |
| :--- | :--- | :--- | :--- | :--- |
| `batch_*` | Handles category wizard, joining, listing | `internal/bot/router.go` + `commands_impl.go` | Yes | Verified structure |
| `admin_*` | Settings wizard and lock toggles | `internal/bot/router.go` + `commands_impl.go` | Yes | Verified structure |
| `demo_*` | Invokes 3-hour timer for paid batches | `internal/bot/router.go` | Yes | State persistence verified via Scheduler |
| `confirm_*` | Confirmation steps for destructive admin actions | Mapped via transpiler | Yes | Handled synchronously |
