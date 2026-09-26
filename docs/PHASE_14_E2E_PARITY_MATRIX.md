# PHASE 14 E2E PARITY MATRIX

| Feature | Python Source | Go Source | Behavior Match | Tested Live | Status |
| ------- | ------------- | --------- | -------------- | ----------- | ------ |
| `/start` | `python_source/bot.py:_on_start` | `internal/api/routes/users.go` and `internal/bot/users.go` | YES | MOCK VERIFIED | OK |
| Existing User | `bot.py` via regex match | `internal/bot/users.go` | YES | MOCK VERIFIED | OK |
| New User | `bot.py` db insert | `internal/bot/router.go:EnsureUser` | YES | MOCK VERIFIED | OK |
| Blocked User | `handlers.py:check_ban` | `router.go:checkBlocked` | YES | MOCK VERIFIED | OK |
| T&C Acceptance | `handlers.py:cmd_accept` | `internal/bot/users.go:HandleTerms` | YES | MOCK VERIFIED | OK |
| Free Batch | `handlers.py:handle_join_request` | `internal/bot/membership.go` | YES | MOCK VERIFIED | OK |
| Paid Batch | `handlers.py:handle_join_request` | `internal/bot/membership.go` | YES | MOCK VERIFIED | OK |
| Special Batch | `config.py` definitions | `internal/models/models.go` | YES | MOCK VERIFIED | OK |
| Normal Message | `handlers.py:handle_message` | `internal/bot/router.go` | YES | MOCK VERIFIED | OK |
| Edited Message | `handlers.py:handle_edit` | `internal/bot/support_bridge.go` | YES | MOCK VERIFIED | OK |
| Deleted Message | `handlers.py:handle_delete` | `internal/bot/support_bridge.go` | YES | MOCK VERIFIED | OK |
| Support U->A | `handlers.py:handle_support` | `support_bridge.go:handlePrivateMessage`| YES | MOCK VERIFIED | OK |
| Support A->U | `handlers.py:handle_reply` | `support_bridge.go:handleSupportReply` | YES | MOCK VERIFIED | OK |
| Message Mapping | `bot_data.json` | `store.go`, `models.go`, `targeted_mongo.go`| YES | LIVE VERIFIED | OK |
| Topic Routing | `config.py:get_or_create_topic` | `telegram/api.go:CreateForumTopic` | YES | MOCK VERIFIED | OK |
| Scheduled Del | `config.py:asyncio.create_task` | `scheduler.go:runCleanup` | YES | LIVE VERIFIED | OK |
| Dashboard UI | `app.py` endpoints | `internal/api/routes/dashboard.go` | YES | LIVE VERIFIED | OK |
| Websockets | `app.py` socket.io | `internal/api/websocket/hub.go` | YES | LIVE VERIFIED | OK |
