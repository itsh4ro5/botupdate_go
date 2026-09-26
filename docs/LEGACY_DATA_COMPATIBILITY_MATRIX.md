# LEGACY DATA COMPATIBILITY MATRIX
*Systematic Python -> Go Type Mismatch Audit*

| Field | Python Type | Existing DB Type | Go Type | Compatible | Adapter Required | Risk |
| ----- | ----------- | ---------------- | ------- | ---------- | ---------------- | ---- |
| `admin_ids` | `list` | JSON Array / BSON Array | `map[int64]struct{}` | ❌ No | **YES** (`compat.go` implemented) | P0 (Boot Panic) |
| `blocked_users` | `list` | JSON Array / BSON Array | `map[int64]struct{}` | ❌ No | **YES** (`compat.go` implemented) | P0 (Boot Panic) |
| `users` | `dict` (keys: str) | String-keyed Object/Document | `map[int64]*User` | ⚠️ Partial | N/A (Standard Go/BSON decoder handles string keys parsing to Int64 gracefully) | P1 |
| `free_batches` | `dict` (keys: str) | String-keyed Object/Document | `map[int64]*Batch` | ⚠️ Partial | N/A | P1 |
| `paid_batches` | `dict` (keys: str) | String-keyed Object/Document | `map[int64]*Batch` | ⚠️ Partial | N/A | P1 |
| `message_map` | `dict` (str -> str) | String-keyed Object/Document | `map[string]string`| ✅ Yes | N/A | P3 |
| `user_topics` | `dict` (str -> obj)| String-keyed Object/Document | `map[int64]*SupportTopic`| ⚠️ Partial | N/A | P1 |
| `pending_requests` | `dict` | String-keyed Object/Document | `map[string]*PendingRequest`| ✅ Yes | N/A | P3 |

### Safe Handling Confirmed
Go's `encoding/json` and `go.mongodb.org/mongo-driver/bson` drivers possess native behavior that automatically translates BSON/JSON string-keys (e.g. `"123456"`) directly into `map[int64]...` types so long as the strings can be parsed to integers. As such, all primary `Users` and `Batches` maps are inherently backwards-compatible without requiring custom unmarshaling logic. 

The ONLY strictly incompatible fields were the lists/arrays representing sets (`admin_ids` and `blocked_users`), which have now been wrapped in a completely backwards-compatible BSON/JSON custom unmarshaler `compat.go`.
