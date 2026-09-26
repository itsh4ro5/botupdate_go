# API PARITY MATRIX: Frontend `api.ts` vs Go Fiber Routes

## 1. Authentication (`/auth`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/auth/me` | `/api/v1/auth/me` | GET | Yes | Matching |
| `/auth/login` | `/api/v1/auth/login` | POST | No | Matching |
| `/auth/logout` | `/api/v1/auth/logout` | POST | Yes | Matching |
| `/auth/change-password` | `/api/v1/auth/change-password` | POST | Yes | Matching |

## 2. Dashboard (`/dashboard`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/dashboard/overview` | `/api/v1/dashboard/overview` | GET | Yes | Matching |

## 3. Users (`/users`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/users` | `/api/v1/users` | GET | Yes | Matching |
| `/users/:id` | `/api/v1/users/:id` | GET | Yes | Matching |
| `/users/:id/block` | `/api/v1/users/:id/block` | POST | Yes (ADMIN) | Matching |
| `/users/:id/unblock` | `/api/v1/users/:id/unblock` | POST | Yes (ADMIN) | Matching |
| `/users/:id/tier` | `/api/v1/users/:id/tier` | POST | Yes (OWNER) | Matching |

## 4. Admins & Roles (`/admins`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/admins` | `/api/v1/admins` | GET | Yes | Matching |
| `/admins` | `/api/v1/admins` | POST | Yes (OWNER) | Matching |
| `/admins/:username/role` | `/api/v1/admins/:username/role` | PUT | Yes (OWNER) | Matching |
| `/admins/:username/sessions/revoke-all` | `/api/v1/admins/:username/sessions/revoke-all` | POST | Yes (OWNER/Self) | Matching |
| `/admins/:username/force-password-change` | `/api/v1/admins/:username/force-password-change` | POST | Yes (OWNER) | Matching |
| `/admins/:username/sessions` | `/api/v1/admins/:username/sessions` | GET | Yes (OWNER/Self) | Matching |
| `/admins/:username/sessions/:id/revoke` | `/api/v1/admins/:username/sessions/:id/revoke` | POST | Yes (OWNER/Self) | Matching |

## 5. Analytics (`/analytics`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/analytics/overview` | `/api/v1/analytics/overview` | GET | Yes | Matching |
| `/analytics/batches` | `/api/v1/analytics/batches` | GET | Yes | Matching |

## 6. Audit (`/audit`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/audit` | `/api/v1/audit` | GET | Yes | Matching |

## 7. Batches (`/batches`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/batches` | `/api/v1/batches` | GET | Yes | Matching |
| `/batches/:id` | `/api/v1/batches/:id` | GET | Yes | Matching |
| `/batches/:id/access/grant` | `/api/v1/batches/:id/access/grant` | POST | Yes | Matching |
| `/batches/:id/access/revoke` | `/api/v1/batches/:id/access/revoke` | POST | Yes | Matching |

## 8. Operations (`/operations`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/operations/system` | `/api/v1/operations/system` | GET | Yes | Matching |
| `/operations/maintenance/refresh-cache` | `/api/v1/operations/maintenance/refresh-cache` | POST | Yes (OWNER) | Matching |

## 9. Requests (`/requests`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/requests` | `/api/v1/requests` | GET | Yes | Matching |
| `/requests/:id/approve` | `/api/v1/requests/:id/approve` | POST | Yes | Matching |
| `/requests/:id/reject` | `/api/v1/requests/:id/reject` | POST | Yes | Matching |

## 10. Support (`/support`)
| Frontend Call | Backend Route | Method | Auth Required | Status |
|---|---|---|---|---|
| `/support/conversations` | `/api/v1/support/conversations` | GET | Yes | Matching |
| `/support/conversations/:id/messages` | `/api/v1/support/conversations/:id/messages` | GET | Yes | Matching |
| `/support/conversations/:id/reply` | `/api/v1/support/conversations/:id/reply` | POST | Yes | Matching |

## Summary
100% of frontend `api.ts` requests map precisely to backend Fiber routes. There are no stale endpoints, missing endpoints, or route permission mismatches between React and Go. Auth context propagates securely via Fiber locals and HttpOnly session cookies.
