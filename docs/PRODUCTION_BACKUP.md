# Production Backup & Recovery Strategy

## What Must Be Backed Up
The Telegram Command Center stores all mission-critical data within a single MongoDB collection defined by the `models.BotState` struct (typically under the key `"state"`). This document contains:
- `WebAdmins` (Usernames, roles, hashed passwords, forced-reset flags)
- `Users` (Telegram user profiles, tier statuses)
- `Batches` (Access grants, paid/free states)
- `PendingRequests` (Join requests waiting for approval)
- `UserTopics` (Support conversations)
- Other bot configuration state.

## How to Back It Up
Use standard MongoDB tools to extract the state safely without blocking active reads/writes.
```bash
mongodump --uri="mongodb+srv://<username>:<password>@<cluster>.mongodb.net/<database>" --collection=state --out=/backup/botupdate_state
```
Since the bot relies on targeted `$set` updates, it is highly recommended to perform backups during periods of low activity.

## Verification
To verify a backup, restore it to an isolated, temporary local MongoDB instance:
```bash
mongorestore --uri="mongodb://localhost:27017/botupdate_test" /backup/botupdate_state
```
Check that the collection contains the single state document and that the admin user hashes match expectations.

## Restoration
**WARNING**: Restoration overwrites the entire BotState. ONLY do this if the live state is corrupted or lost.
1. Stop the active Go backend to prevent conflicting writes during restore.
   ```bash
   kill -SIGTERM $(pgrep botupdate)
   ```
2. Restore the database:
   ```bash
   mongorestore --uri="mongodb+srv://<username>:<password>@<cluster>.mongodb.net/<database>" --collection=state /backup/botupdate_state --drop
   ```
3. Restart the Go backend.

## What NOT to do
- Do NOT perform manual manual database migrations.
- Do NOT alter `WebAdmins` directly in the database to bypass password hashes (use the `/force-password-change` endpoint instead).
- Do NOT attempt to split the BotState into multiple relational collections (the Go backend relies heavily on the single-document memory sync).

## Recovery Checklist
- [ ] Backend process stopped successfully?
- [ ] Database restored without errors?
- [ ] Go backend restarts without panic?
- [ ] Owner login succeeds?
- [ ] Dashboard displays accurate historical user counts?
- [ ] Telegram bot responds to `/start`?
