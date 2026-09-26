# Production Runbook

## 1. Initial Deployment
The application compiles into a statically linked Go binary and a set of static HTML/JS/CSS files (Vite build output).
- Build Backend: `go build -o botupdate ./cmd/bot`
- Build Frontend: `cd web && npm install && npm run build` (host `/dist` statically or via a CDN).

## 2. Environment Variables
Ensure the following variables are securely injected at runtime (do NOT commit to `.env`):
- `MONGO_URL`: MongoDB connection string.
- `TELEGRAM_BOT_TOKEN`: Telegram bot token.
- `API_ID` & `API_HASH`: MTProto credentials.
- `WEB_ADMIN_USERNAME` & `WEB_ADMIN_PASSWORD`: Used strictly for the *first* initial boot to bootstrap the OWNER account. Once created, they are ignored in favor of the hashed DB entry.
- `PORT`: (Optional) Port for the web API (defaults to 3000).

## 3. Starting Backend
Run the compiled binary:
```bash
./botupdate
```

## 4. Starting Frontend/Static Assets
The frontend artifacts in `web/dist` should be served by Nginx, Caddy, or a managed static host (Vercel, Netlify). 

## 5. MongoDB Dependency
MongoDB is required for persistent state and acts as the sole source of truth. If MongoDB goes down, the backend operates in a degraded mode and the HTTP API will return 503 errors on the readiness endpoint.

## 6. Health Endpoint
- **URL**: `GET /api/v1/health`
- **Purpose**: Liveness check. Returns 200 OK immediately if the HTTP server is listening.

## 7. Readiness Endpoint
- **URL**: `GET /api/v1/ready`
- **Purpose**: Checks active database connectivity. Returns 503 if the database is unreachable.

## 8. Authentication
Authentication utilizes securely hashed bcrypt passwords and server-side stateful tracking in the database (`WebSessions`). Role assignments (`OWNER`, `ADMIN`, `SUPPORT`) are strictly validated at the middleware layer.

## 9. WebSocket Behavior
The WebSocket hub requires authentication *prior* to protocol upgrade. A non-blocking buffered drop mechanism prevents slow browser clients from locking the global event publisher.

## 10. Log Inspection
Check stdout/stderr logs. The application provides detailed structured API latency and method logs, alongside specific subsystem error strings.

## 11. Graceful Shutdown & Restart
Send a standard interrupt:
```bash
kill -SIGTERM <PID>
```
The application will safely conclude pending Telegram handler goroutines (via `sync.WaitGroup`) and allow up to 5 seconds for in-flight HTTP REST handlers to finish before terminating.

## 12. Rollback Procedure
If a new release causes critical issues:
1. Revert to the previous Go binary.
2. Restart the service.
3. No database rollbacks are generally required as schema migrations are prohibited.

## 13. Common Failure Symptoms
- **Constant 503s on `/ready`**: Verify MongoDB URI and database whitelist IPs.
- **WebSocket dropping constantly**: Usually indicative of reverse-proxy timeout constraints (ensure Nginx `proxy_read_timeout` is sufficient).
- **Stuck Telegram Updates**: Ensure there is no competing webhook configured for the bot token.

## 14. Telegram Regression Checks
After any deployment, verify:
- Bot responds to `/start`.
- Support messages route correctly between the web dashboard and Telegram group.
- Batch generation and callback query buttons process accurately.
