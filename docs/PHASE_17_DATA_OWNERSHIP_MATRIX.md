# PHASE 17: DATA OWNERSHIP MATRIX

| Data Field | Authoritative Source | Read by Python | Read by Go | Written by Python | Written by Go | Shared? | Safe? |
| ---------- | -------------------- | -------------- | ---------- | ----------------- | ------------- | ------- | ----- |
| **`admin_ids`** | MongoDB | YES | YES | YES | YES | YES | **YES** (Via custom compat adapter) |
| **`blocked_users`** | MongoDB | YES | YES | YES | YES | YES | **YES** (Via custom compat adapter) |
| **`users`** | MongoDB | YES | YES | YES | YES | YES | **YES** |
| **`free_batches`** | MongoDB | YES | YES | YES | YES | YES | **YES** |
| **`paid_batches`** | MongoDB | YES | YES | YES | YES | YES | **YES** |
| **`user_topics`** | MongoDB | YES | YES | YES | YES | YES | **YES** |
| **`pending_requests`**| MongoDB | YES | YES | YES | YES | YES | **YES** |
| **`message_map`** | MongoDB | YES | YES | YES | YES | YES | **YES** |
| **`web_admins`** | MongoDB | NO | YES | NO | YES | NO | **YES** (Go exclusive RBAC) |
| **`web_sessions`** | MongoDB | NO | YES | NO | YES | NO | **YES** (Go exclusive auth) |
| **Flashcard Data** | Firebase | YES | NO | YES | NO | NO | **YES** (Safely ignored) |

### Ownership Conclusion
Go uses `$set` semantics targeting explicitly mapped paths (e.g., `data.users`). Go **does not** own unmapped fields. Python owns any undocumented legacy variables, and they remain entirely untouched in the database because Go only updates what it reads and mutates.
