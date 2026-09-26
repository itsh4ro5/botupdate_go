# FINAL API CONTRACT MATRIX
*React Frontend (`api.ts` & component calls) vs Fiber Backend*

| Endpoint | Method | Component / Caller | Backend Route (`internal/api/routes`) | Authentication | Role Required | Status |
| -------- | ------ | ------------------ | ------------------------------------- | -------------- | ------------- | ------ |
| `/auth/me` | GET | `AuthContext.tsx` | `auth.go` | Token/Session | Any | ✅ Perfect |
| `/auth/login` | POST | `Login.tsx` | `auth.go` | None | N/A | ✅ Perfect |
| `/auth/logout` | POST | `AuthContext.tsx` | `auth.go` | Token/Session | Any | ✅ Perfect |
| `/auth/change-password` | POST | `ChangePassword.tsx`, `SecuritySettings.tsx` | `auth.go` | Token/Session | Any | ✅ Perfect |
| `/dashboard/overview` | GET | `Dashboard.tsx` | `dashboard.go` | Token/Session | Any | ✅ Perfect |
| `/users` | GET | `Users.tsx` | `users.go` | Token/Session | Any | ✅ Perfect |
| `/users/:id` | GET | `Users.tsx` | `users.go` | Token/Session | Any | ✅ Perfect |
| `/users/:id/block` | POST | `Users.tsx` | `users.go` | Token/Session | ADMIN | ✅ Perfect |
| `/users/:id/unblock`| POST | `Users.tsx` | `users.go` | Token/Session | ADMIN | ✅ Perfect |
| `/users/:id/tier` | POST | `Users.tsx` | `users.go` | Token/Session | OWNER | ✅ Perfect |
| `/admins` | GET | `Admins.tsx` | `admin.go` | Token/Session | Any | ✅ Perfect |
| `/admins` | POST | `Admins.tsx` | `admin.go` | Token/Session | OWNER | ✅ Perfect |
| `/admins/:username/role` | PUT | `Admins.tsx` | `admin.go` | Token/Session | OWNER | ✅ Perfect |
| `/admins/:username/sessions` | GET | `SecuritySettings.tsx` | `admin.go` | Token/Session | Self/OWNER | ✅ Perfect |
| `/admins/:username/sessions/:id/revoke` | POST | `SecuritySettings.tsx` | `admin.go` | Token/Session | Self/OWNER | ✅ Perfect |
| `/admins/:username/sessions/revoke-all` | POST | `Admins.tsx`, `SecuritySettings.tsx` | `admin.go` | Token/Session | Self/OWNER | ✅ Perfect |
| `/admins/:username/force-password-change` | POST | `Admins.tsx` | `admin.go` | Token/Session | OWNER | ✅ Perfect |
| `/analytics/overview` | GET | `Analytics.tsx` | `analytics.go` | Token/Session | Any | ✅ Perfect |
| `/analytics/batches` | GET | `Analytics.tsx` | `analytics.go` | Token/Session | Any | ✅ Perfect |
| `/audit` | GET | `Audit.tsx` | `audit.go` | Token/Session | Any | ✅ Perfect |
| `/batches` | GET | `Batches.tsx` | `batches.go` | Token/Session | Any | ✅ Perfect |
| `/batches/:id` | GET | `Batches.tsx` | `batches.go` | Token/Session | Any | ✅ Perfect |
| `/batches/:id/access/grant` | POST | `Batches.tsx` | `batches.go` | Token/Session | ADMIN | ✅ Perfect |
| `/batches/:id/access/revoke` | POST | `Batches.tsx` | `batches.go` | Token/Session | ADMIN | ✅ Perfect |
| `/operations/system` | GET | `Operations.tsx` | `operations.go` | Token/Session | Any | ✅ Perfect |
| `/operations/maintenance/refresh-cache` | POST | `Operations.tsx` | `operations.go` | Token/Session | OWNER | ✅ Perfect |
| `/requests` | GET | `Requests.tsx` | `requests.go` | Token/Session | Any | ✅ Perfect |
| `/requests/:id/approve` | POST | `Requests.tsx` | `requests.go` | Token/Session | ADMIN | ✅ Perfect |
| `/requests/:id/reject` | POST | `Requests.tsx` | `requests.go` | Token/Session | ADMIN | ✅ Perfect |
| `/support/conversations` | GET | `Support.tsx` | `support.go` | Token/Session | Any | ✅ Perfect |
| `/support/conversations/:id/messages` | GET | `Support.tsx` | `support.go` | Token/Session | Any | ✅ Perfect |
| `/support/conversations/:id/reply` | POST | `Support.tsx` | `support.go` | Token/Session | Any | ✅ Perfect |

## Findings
- **Stale endpoints**: None.
- **Missing endpoints**: None.
- **WebSocket Paths**: Correctly mapping `ws://` / `wss://` on `/ws` endpoint with secure token exchange.
- **Hardcoded Localhost**: `BASE_URL` dynamically loads from `import.meta.env.VITE_API_URL` safely.
- **Security**: The Fiber API strictly isolates Admin functions using `c.Locals("user")` without risking `interface conversion panics`. All endpoints have 1:1 parity and return `JSON`.
