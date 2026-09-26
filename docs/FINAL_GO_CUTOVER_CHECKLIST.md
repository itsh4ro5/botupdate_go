# FINAL GO CUTOVER CHECKLIST
**Status: GREEN (PRODUCTION READY)**

## 1. MONGODB BACKUP
- [ ] Log in to MongoDB Atlas / your MongoDB provider.
- [ ] Navigate to the `botupdate` database cluster.
- [ ] Trigger an on-demand snapshot/backup.
- [ ] Verify the snapshot completes successfully.

## 2. PYTHON DEACTIVATION
- [ ] Navigate to the Hugging Face Space (or VPS) currently hosting the Python `app.py` bot.
- [ ] Go to Settings -> Pause Space (or stop the systemd service).
- [ ] Wait exactly 60 seconds to ensure the Telegram Long Polling webhook drops cleanly and no background `asyncio` tasks are mutating the database.

## 3. GO DEPLOYMENT STARTUP
- [ ] Ensure `.env` is populated correctly on the new production server with `MONGO_URI`, `BOT_TOKEN`, and `JWT_SECRET`.
- [ ] Start the Go binary `./botupdate`.
- [ ] Verify console outputs:
  ```text
  Bot authorized successfully
  MongoDB connected
  Websocket Hub started
  Fiber server listening on port...
  ```

## 4. TELEGRAM CONNECTIVITY VERIFICATION
- [ ] As a regular User, send `/start` to the Telegram bot.
- [ ] Verify the bot responds instantly with the welcome menu.
- [ ] As an Admin/Owner, send `/admin` to verify RBAC access.

## 5. COMMAND CENTER VERIFICATION
- [ ] Log in to the React Web application using your existing credentials.
- [ ] Navigate to the `Dashboard` and verify counts match legacy expectations.
- [ ] Navigate to `Users` and `Batches` to verify the JSON array parsing successfully decoded legacy Python fields.

## 6. SUPPORT VERIFICATION (2-WAY)
- [ ] From Telegram, send a message to the bot.
- [ ] Open the React Command Center `Support` tab. 
- [ ] Verify the user message appears on the **left side** of the UI.
- [ ] Send a reply from the **right side** of the UI (Admin).
- [ ] Verify the reply arrives successfully in the user's Telegram DM.

## 7. DATABASE WRITE VERIFICATION
- [ ] Trigger an action (e.g. create a batch, or approve a request).
- [ ] Check MongoDB explicitly to confirm Go used targeted `$set` (e.g. `data.batches`) and did not destroy unrelated legacy fields (e.g. `data.legacy_flashcards`).

## 8. MONITOR LOGS
- [ ] Tail the Go application logs for 15 minutes.
- [ ] Check for `panic` or `c.Locals interface conversion` faults.
- [ ] Go operates seamlessly without throwing random stack traces.
