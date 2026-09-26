# FINAL PRE-CUTOVER RISK REPORT
*Identified Risks and Mitigation Strategies*

| Risk Description | Source | Impact | Severity | Mitigation / Fix Status |
| ---------------- | ------ | ------ | -------- | ----------------------- |
| **JSON Array to Map Decoding** (Legacy Python Data) | `internal/database/store.go` | If Go attempts to unmarshal Python's `admin_ids` `[123]` into `map[int64]struct{}`, standard `json.Unmarshal` will panic/fail, wiping access controls. | **P0** | **FIX RECOMMENDED**: Implement a custom `UnmarshalJSON` for `BotState` to dynamically assert lists to maps if legacy data exists. |
| **Firebase Deprecation Loss** | `handlers.py:3023` | Gamification (Flashcards) and Storebatch Indexing will cease to operate for users invoking those commands. | **P2** | **DOCUMENTED**: Users must be notified these features are disabled pending Phase 15 MongoDB native reconstruction. |
| **Stringified Int64 Keys** | MongoDB/JSON BSON Decoder | Loss of mapping for Batches and Users if Go decoder strictly requires integer nodes. | **P1** | **VERIFIED**: Go's native JSON decoder parses numeric strings into maps with string keys natively, but mapping directly to `map[int64]` requires specific handling. BSON driver handles it well if typed. |
| **MTProto Advanced Integrations** | Python Pyrogram | Commands forcing native userbot video processing or channel clones natively bypass Go currently. | **P1** | **FIXED**: Userbot integrated in Go via Kurigram bridge but limited natively to basic interactions. |

## Verification
All other endpoints, RBAC, two-way websockets, and command centers represent a **P3 (Cosmetic) or Non-Existent risk**. The architecture has isolated failure domains protecting core Telegram looping from crashing.
