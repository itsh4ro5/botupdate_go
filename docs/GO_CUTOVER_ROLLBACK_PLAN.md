# GO CUTOVER & ROLLBACK PLAN
*Safe Migration and Reversal Strategy*

## Stage 1: Pre-Cutover Preparation
1. **Disable Go Polling**: Ensure the Go application `.env` has polling temporarily disabled or uses a distinct test bot token.
2. **Snapshot Production**: Execute a full MongoDB export `mongodump --uri="<PROD_URI>"` and download the Firebase Firestore backup.
3. **Database Pre-Flight Read**: Boot the Go application. Verify it reads the JSON/BSON structures correctly without panic (specifically checking `users` and `admin_ids` list-to-map translations).

## Stage 2: Cutover Execution
1. **Stop Python Bot**: Gracefully terminate the Python `bot.py` process on Hugging Face to stop consuming Webhooks/Long Polling.
2. **Start Go Bot**: Launch the Go Application `./botupdate` with the production `BOT_TOKEN`. 
3. **Verify Health**: Immediately check the Go console for:
   - "Bot authorized successfully"
   - "Scheduler started"
   - No `json: cannot unmarshal array into Go struct field` errors.

## Stage 3: Rollback Plan (If Go Fails)
If the Go application crashes, panics, or corrupts mappings within the first 12 hours:
1. **Stop Go Process**: Immediately kill the Go binary/container.
2. **Assess Database Contamination**: Since Go uses targeted `$set` modifiers, it is highly likely the database is still fully compatible with Python.
3. **Restart Python Bot**: Bring the Hugging Face python bot back online. Python's loose typing will simply ignore any strictly typed fields Go might have added, reading its legacy JSON arrays normally.
4. **If Data is Severely Corrupted**: Restore the MongoDB collection using `mongorestore` from the backup taken in Stage 1. Re-launch Python.

**CRITICAL RULE**: Do NOT run Python and Go with the same `BOT_TOKEN` simultaneously. This causes `Conflict: terminated by other getUpdates request` and drops Telegram messages entirely.
