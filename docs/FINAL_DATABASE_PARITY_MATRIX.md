# FINAL DATABASE PARITY MATRIX
*Pre-Cutover Forensic Analysis: Python vs Go MongoDB Schema Compatibility*

## MongoDB Document Structures

| Python Storage | Python Field | Type | Go Field | Go Type | Compatible | Risk |
| -------------- | ------------ | ---- | -------- | ------- | ---------- | ---- |
| `bot_data` | `admin_ids` | `list` of `int` | `AdminIDs` | `map[int64]struct{}` | ⚠️ Partial | P1 (Must parse JSON array to map upon load) |
| `bot_data` | `free_batches` | `dict` (str -> dict) | `FreeBatches` | `map[int64]*Batch` | ⚠️ Partial | P1 (String keys vs int64 keys) |
| `bot_data` | `paid_batches` | `dict` (str -> dict) | `PaidBatches` | `map[int64]*Batch` | ⚠️ Partial | P1 (String keys vs int64 keys) |
| `bot_data` | `users` | `dict` (str -> dict) | `Users` | `map[int64]*User` | ⚠️ Partial | P0 (Critical user map keying mismatch) |
| `bot_data` | `categories` | `list` of `str` | `Categories` | `[]string` | ✅ Yes | P3 |
| `bot_data` | `blocked_users`| `list` of `int` | `BlockedUsers` | `map[int64]struct{}` | ⚠️ Partial | P1 (Array to Map conversion required) |
| `bot_data` | `pending_requests` | `dict` | `PendingRequests` | `map[string]*PendingRequest` | ✅ Yes | P2 (Verify inner types) |

### Incompatibility Notes & Safety
- **String vs Int64 Mismatch**: Python JSON automatically coerces integer dictionary keys into strings (e.g. `"123456": { ... }`). Go strongly types `Users` and `Batches` as `map[int64]...`. The Go `JSONStore` decoder currently attempts to natively unmarshal into `int64`. If `json.Unmarshal` fails parsing string keys into `int64` maps directly (which Go's standard library actually supports under the hood gracefully in recent versions), this could wipe out user access. **Verification Required:** Ensure the decoder gracefully handles string-to-int64 unmarshaling without throwing errors.
- **Lists vs Maps (Set representation)**: Python's `admin_ids` and `blocked_users` are `list`. Go defines them as `map[int64]struct{}`. Standard `json.Unmarshal` will fail to unmarshal a JSON array `[1, 2, 3]` into a Go map `{"1": {}, "2": {}, "3": {}}`. **This is a P0 schema migration incompatibility that must be adapter-parsed.**
