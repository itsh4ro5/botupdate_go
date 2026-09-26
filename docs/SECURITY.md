# Security Architecture & Operations

## Authentication Model
The Command Center employs a stateful server-side authentication layer using explicit API calls to create and revoke sessions. Passwords are asynchronously hashed utilizing `bcrypt.DefaultCost`. The system explicitly prevents clear-text transmission outside of initial login payload boundaries. Forced password resets strictly block access to core features until resolved.

## Authorization Model
Role-Based Access Control (RBAC) is enforced exclusively at the Fiber routing middleware layer.
- **OWNER**: Absolute administrative override. Allowed to promote/demote other users and enforce system-wide session terminations.
- **ADMIN**: Basic operations clearance (managing batches, interacting with standard configuration limits).
- **SUPPORT**: Lowest-tier access reserved strictly for conversational routing inside the Support module.

Frontend UI hiding is employed purely for UX hygiene; it is never treated as a security boundary.

## Session Model
Sessions are issued as time-limited tokens stored safely in the database (`WebSessions`). They evaluate automatically against an internal heartbeat mechanism that expires stale authentications dynamically, enforcing hard logouts.

## Secret Handling
- Passwords are NEVER logged.
- The `.env` overrides are explicitly isolated from the compiled Vite artifacts.
- The Bot Token and MTProto credentials remain encapsulated inside the Go core.

## WebSocket Security
The WebSocket connection mandates a valid, non-expired Auth token in order to negotiate the initial handshake upgrade. Malformed or unauthenticated socket upgrade attempts are summarily refused via `401 Unauthorized` blocks.

## API Security
All mutations are strictly bound by the `RequireRole()` and `RequireAuth()` middleware wrappers. Cross-Site Request Forgery (CSRF) tokens and strict Cross-Origin Resource Sharing (CORS) rules govern HTTP mutations safely.

## Operational Security Recommendations
1. Place the backend API behind an Nginx or Caddy layer managing valid SSL/TLS certificates.
2. Restrict MongoDB Atlas inbound connections to strictly the IP footprint of the backend API server.
3. Configure frontend build tools (Vite) strictly without injected VITE_ backend secrets.
