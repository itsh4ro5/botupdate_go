# PHASE 17: PRODUCTION CUTOVER RUNBOOK

## Pre-Requisites
1. Ensure the Python `HF_Space` deployment is accessible and can be paused/stopped.
2. Confirm the exact MongoDB Atlas connection string is mapped to the Go Application `.env` via `MONGO_URI`.

## Phase 1: Snapshot and Python Shutdown
1. Run MongoDB backup through Atlas UI (Snapshot A).
2. Pause/Stop the Hugging Face Python container.
3. Wait exactly 60 seconds to ensure any pending webhooks or polling offsets are flushed and no concurrent writes are operating.

## Phase 2: Go Startup (Read-Only Safety)
1. Run `./botupdate` locally or deploy to the production container.
2. Observe startup logs. Confirm you see:
   - `MongoDB Connected Successfully`
   - `Bot authorized successfully`
   - `Scheduler started`
3. Verify no `Unmarshal` errors occurred during the state load phase.

## Phase 3: Live Verification
1. As an admin, log in to the React Web Command Center.
2. Navigate to `/users` and `/batches` to confirm data is visible.
3. Send a test message to the Bot on Telegram.
4. Verify the message arrives in the Command Center `/support` queue.
5. Reply via the Command Center and verify the message reaches the Telegram user.

## Phase 4: Rollback Strategy (If Go Fails)
If the Go application exhibits unexpected failures or crashes within the first 12 hours:
1. Immediately stop the Go binary/container.
2. Since Go strictly uses `$set` and safely reserializes legacy arrays (e.g. `admin_ids`), no MongoDB restoration is strictly required.
3. Restart the Hugging Face Python container.
4. Python will natively re-read the state safely and resume operations precisely where Go left off.
