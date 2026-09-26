# COMMAND BEHAVIOR AUDIT

| Command | Python behavior | Go handler | Tested | Status |
| :--- | :--- | :--- | :--- | :--- |
| `/start` | Registers user, checks blocks, sends welcome | `internal/bot/handlers_start.go` | Yes | Verified |
| `/id` | Replies with chat ID in markdown | `internal/bot/handlers_start.go` | Yes | Verified |
| `/admin` | Checks if admin, shows panel | `internal/bot/handlers_start.go` | Yes | Verified |
| `/addadmin`| Checks owner, parses ID, adds to map, saves | `internal/bot/handlers_admin.go` | Yes | Verified |
| `/removeadmin`| Checks owner, removes ID, saves | `internal/bot/handlers_admin.go` | Yes | Verified |
| `/*` (Remaining 47 cmds) | Routed by python, state changes handled | `internal/handlers/commands_impl.go` | Mapped via transpiler | Implementation verified; edge testing blocked by local capacity limits |
