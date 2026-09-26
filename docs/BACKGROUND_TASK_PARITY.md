# BACKGROUND TASK PARITY
*Scheduler & Background Operations Audit*

| Python Task (`bot.py` / `app.py`) | Go Equivalent (`scheduler.go`) | Status | Persistence Req | Failure Behavior |
| --------------------------------- | ------------------------------ | ------ | --------------- | ---------------- |
| `_debounced_flush` | N/A (Direct MongoDB `$set` updates) | ✅ Safe | N/A | Fails securely per request |
| `auto_kick_blocked` | `s.UniversalKick` (Mandatory sync) | ✅ Parity | Reads `BlockedUsers` | Gracefully skips chat |
| `_delayed_delete` | `state.ScheduledDeletes` execution | ✅ Parity | YES (Saved to DB) | Ignores/cleans upon failure |
| Demo Expirations | `runCleanup()` Demo loop | ✅ Parity | YES (DB timestamps) | Backoff delays |
| `_background_kick` | Handled via API Ban/Unban requests | ✅ Parity | N/A | Standard backoff |

## Conclusion
The Go `scheduler.go` perfectly mirrors the asynchronous delayed operations of the original Python script. By relying on `context.Context` tied to the Fiber server lifecycle, the scheduler avoids infinite busy loops and gracefully stops when the process receives an interrupt signal.
