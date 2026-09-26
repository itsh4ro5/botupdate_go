# BOTSTATE MIGRATION MAP
*Field-by-Field Authoritative Source Classification*

| Field Name | Type | Classification | Source of Truth | Notes |
| ---------- | ---- | -------------- | --------------- | ----- |
| `AdminIDs` | `map` | **PERSISTENT** | MongoDB (`main_settings.data.admin_ids`) | Requires JSON array decoding bridge. |
| `Users` | `map` | **PERSISTENT** | MongoDB (`users` collection or unified doc) | Core mapping for user tiers and auth. |
| `FreeBatches` | `map` | **PERSISTENT** | MongoDB | Defines tier access limits. |
| `PaidBatches` | `map` | **PERSISTENT** | MongoDB | Defines tier access limits. |
| `BlockedUsers` | `map` | **PERSISTENT** | MongoDB | Essential for ban evasion protection. |
| `UserTopics` | `map` | **PERSISTENT** | MongoDB | Matches Users to Forum Thread IDs. |
| `MessageMap` | `map` | **PERSISTENT** | MongoDB | Vital for 2-way Edit/Delete syncs. |
| `PendingRequests` | `map`| **PERSISTENT** | MongoDB | Must survive restart to approve older requests. |
| `ScheduledDeletes`| `array`| **PERSISTENT** | MongoDB | Prevent spam messages from becoming permanent. |
| `WebSessions` | `map` | **RUNTIME** | MongoDB (Transient) | Expires via TTL; safe to clear but ideally persists. |
| `WebAdmins` | `map` | **PERSISTENT** | MongoDB | Contains hashed passwords and 2FA secrets. |
| `MaintenanceMode` | `bool` | **PERSISTENT** | MongoDB | Global lock. |

### Derived / Telegram-Sourced Data
- **Topic IDs**: While stored in MongoDB (`UserTopics`), the original thread is owned by Telegram. If the Thread is deleted on Telegram, the Go Bot must gracefully `unset` the MongoDB record and recreate it.
- **Message IDs**: Purely Telegram-sourced. The bot trusts Telegram entirely for message routing offsets.

### Firebase Sourced
- **None**: No core BotState structs natively depend on Firebase. All dependencies are decoupled.
