# Phase 13 — Critical Bug Fix & Support Stabilization Report

## 1. Critical Bugs Found
1. **Admin Panic**: Navigating to Admin and other routes without properly formed authentication state caused a critical process panic: `panic: interface conversion: interface {} is nil, not string`. This was due to `RequireAuth` missing the `username` binding in `c.Locals`, combined with unsafe type assertions throughout `internal/api/routes`.
2. **WebSocket 1005 (CloseNoStatusReceived)**: Repeated error logs for WS 1005. It was a normal browser navigation disconnect triggered by React unmounting/re-mounting or user page reload.
3. **Support Interface Incomplete**: The Support console was not fully behaving as a 2-way system; payloads were untyped, unstructured, and the frontend relied on old mapping keys.
4. **Missing Fiber Panic Recovery**: The server lacked a global recovery middleware, meaning any unhandled panic would instantly crash the entire production process.

## 2. Bugs Fixed
1. **Safe Type Assertions**: Replaced all instances of unsafe `c.Locals("key").(string)` with a `getStringLocalSafe(c, key)` helper that gracefully handles missing or improperly typed context variables and returns HTTP 401 Unauthorized instead of panicking.
2. **Middleware Context Updates**: Fixed `internal/api/middleware/auth.go` to explicitly bind `username` and `session_id` into `c.Locals`.
3. **WS 1005 Suppression**: Downgraded 1005 (CloseNoStatusReceived) to be recognized as an expected client disconnect in `internal/api/ws/handler.go` utilizing `websocket.IsUnexpectedCloseError` to filter it out.
4. **Fiber Panic Recovery**: Added `recover.New()` at the top of the Fiber middleware stack in `internal/api/server.go` to trap and log future panics safely without taking down the server.

## 3. Support Architecture Redesign
- **Telegram → Go → EventBus → WebSocket → React**: Private messages caught by `handlePrivateMessage` in `support_bridge.go` now publish a strongly-typed `models.SupportMessage` containing `ConversationID`, `SenderType="incoming"`, `TelegramMsgID`, and `Text`. The UI catches this via `e.detail.data.message` and appends it dynamically with auto-scroll.
- **React → API → Go → Telegram**: The `Support.tsx` UI sends a POST request to `/api/v1/support/conversations/:id/reply`. The `SupportService.Reply` sends it to the user, copies it to the support group for state syncing, and emits an "outgoing" `models.SupportMessage` via EventBus. The frontend suppresses duplicate optimistic messages.
- The UI mimics a live chat console. Historical messages are purposefully not faked; instead, a clear banner indicates that history is not stored in memory and live messages will appear below it.

## 4. Verification Tests
- **Go Build**: PASS
- **Go Tests**: PASS (All cached and green)
- **Race Tests**: PASS (`go test -race ./...`)
- **Go Vet**: PASS
- **npm Build**: PASS (Vite production build successful)
- **Playwright Test Suite**: PASS (Executed `web/test_phase13.cjs`)
  - Desktop authentication (Login / Dashboard)
  - Desktop WebSocket connections (101 Switching Protocols verified)
  - Desktop Admin page (Confirmed backend does not crash on load)
  - Desktop / Mobile Support Chat UI (Confirmed responsive views render correctly)
- **API Status**: HTTP 200/401 properly managed, 0 server crashes.

## 5. Screenshots Generated
- `browser_tests/phase13_dashboard_desktop.png`
- `browser_tests/phase13_admin_stable.png`
- `browser_tests/phase13_support_desktop.png`
- `browser_tests/phase13_support_mobile.png`

## 6. Telegram Regression Verification
- **Polling & Handlers**: INTACT (No new polling loops added).
- **`/del` and Reactions**: INTACT (Preserved `msgMap` keys in `support_bridge.go`).
- **Callbacks & Batch Access**: INTACT.
- **Support Bridge**: INTACT (Message copying to Support Group perfectly maintained during bot replies).

## 7. Remaining Limitations
- **Historical Support Messages**: Since Telegram messages are not persisted to a MongoDB collection historically, reloading the page clears the Support UI history. This is by design to avoid faking data, as explicitly directed.

**FINAL RELEASE DECISION**: STABLE. All requirements achieved. No regressions introduced.
