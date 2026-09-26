# EMERGENCY ROLLBACK PROCEDURE (GO -> PYTHON)

If the Go backend demonstrates unrecoverable anomalies or panic loops in production, you can instantly revert back to the legacy Python implementation with zero data loss.

## WHY IT IS SAFE
Because the Go backend utilizes explicit `$set` and `$unset` BSON directives (via `safeFlatten`), it **does not** overwrite or erase the legacy Python schema. All fields that Go did not natively manage (e.g. Gamification elements) remained fully preserved in MongoDB. Furthermore, Go specifically downcasts its `map[int64]struct{}` back to strict JSON/BSON Arrays (`[]int64`) inside `compat.go` upon saving, ensuring Python iterates natively without Type Errors.

## STEP-BY-STEP ROLLBACK

### 1. HALT GO 
- Immediately kill the `./botupdate` process.
- Confirm the process has stopped to release the Telegram Long-Polling lock. (Wait 30-60 seconds for Telegram API to clear `Conflict: terminated by other getUpdates`).

### 2. NO MONGODB RESTORE NECESSARY
- Assuming no malicious manipulation outside of normal operations occurred, the MongoDB data structure is fully backwards compatible right now. You DO NOT need to restore the snapshot unless you explicitly identify data corruption.

### 3. BOOT PYTHON
- Navigate to your Hugging Face Space (or VPS).
- Resume / Start the Python `app.py` script.
- Python will instantly fetch the `main_settings` document, load the `ADMIN_IDS` arrays cleanly, parse its users, and resume polling exactly where Go stopped.

### 4. VERIFY RECOVERY
- Execute a `/ping` command against the bot to verify the Python handler captures it.
- Your rollback is complete.
