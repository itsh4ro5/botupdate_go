# PHASE 17: SAFE FLATTEN AUDIT

## Analysis of `internal/database/safe_flatten.go`

### 1. Which fields are flattened?
All root-level fields of `models.BotState` are flattened. A field like `AdminIDs` becomes `"data.admin_ids"` in the BSON `$set` document.

### 2. Which fields are ignored?
Any fields inside `main_settings.data` in MongoDB that are NOT explicitly defined in Go's `models.BotState` struct are ignored during unmarshaling and thus completely excluded from the `$set` payload. This guarantees their safety (they are never overwritten).

### 3. Which fields are explicitly unset?
None. `safeFlatten` only produces a `$set` payload. It never emits `$unset`.

### 4. What happens to nil fields?
If a map or slice in the struct is `nil`, it marshals to BSON `null` and will overwrite the existing database field with `null`.

### 5. What happens to empty maps?
Empty maps marshal to an empty BSON Document `{}`.

### 6. What happens to empty arrays?
Empty arrays marshal to an empty BSON Array `[]`.

### 7. What happens to unknown Python fields?
They are **safely preserved**. Because Go uses targeted `$set` with explicit dot-notation (e.g., `{"$set": {"data.users": ...}}`), sibling keys that Go knows nothing about are left untouched by MongoDB.

### 8. What happens to nested structures?
Go's `$set` replaces the **entire value** of the field. For example, `$set: {"data.users": ...}` completely overwrites the entire `users` tree. Any unknown fields nested *inside* `data.users` would be lost.

### 9. What happens to BSON numeric types?
Handled safely by the MongoDB Go driver. Int64s remain Int64s.

### 10. What happens to timestamps?
`time.Time` fields are serialized safely into standard BSON datetime primitives.

### 11. What happens to MongoDB ObjectIDs?
Go does not mutate `_id`. The document ID is strictly bound to `"main_settings"`.

### 12. Can a Go save accidentally overwrite a Python-owned field?
**YES (P1)**. If Python injects a new field inside `data.users.1234.new_python_flag`, when Go loads that User and resaves it, Go will overwrite `data.users.1234` entirely with its known schema, erasing `new_python_flag`.

### 13. Can a Go save accidentally convert an array into an empty map?
**YES (P0 BLOCKER)**. Legacy Python saves `admin_ids` and `blocked_users` as Lists (`[]`). Go uses `compat.go` to safely *read* these arrays into Maps. However, when Go *saves* the state, standard BSON marshaling writes the Map back as a BSON Document (`{}`). If a rollback to Python occurs, Python expects a List but receives a Dictionary. Iterating this Dictionary in Python yields String keys, breaking `if user.id in admin_ids` checks completely!

### 14. Can a Go save accidentally remove legacy Firebase-related state?
**NO**. Firebase is physically hosted on an entirely separate infrastructure (Firestore). Go's MongoDB queries cannot affect it.
