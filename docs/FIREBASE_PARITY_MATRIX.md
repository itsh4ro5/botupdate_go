# FIREBASE PARITY MATRIX
*Pre-Cutover Forensic Analysis: Firebase Dependencies*

## Firebase Read/Write Behaviors

| Feature | Python Implementation | Go Implementation | Required for Core Runtime? | Status | Risk |
| ------- | --------------------- | ----------------- | -------------------------- | ------ | ---- |
| Daily Flash Challenge | `app.py` exposes `/api/flash/current` and `POST /api/flash/generate` writing to Firestore. | **MISSING** | No (Gamification feature) | ⚠️ Missing | P2 (Cosmetic feature drop) |
| StoreBatch Data Indexing | `handlers.py:3023` allows indexing older videos/PDFs from a channel and dumping IDs to Firestore for fast retrieval. | **MISSING** | No (Ancillary search feature) | ⚠️ Missing | P1 (Feature loss) |
| User Profile Syncing | `handlers.py` attempts a fallback `FIREBASE SAVE LOGIC` for some generic profile stats. | **MISSING** | No (Purely diagnostic/redundant) | ✅ Safely Ignored | P3 |

## Conclusion & Safety Check
Firebase is **not** used as the primary source of truth for Authentication, Batch Memberships, Event Routing, or Support Topics. Its responsibilities in the Python codebase are strictly segregated into **Gamification** (Flashcards) and **File Indexing** (Storebatch search). 

Because the Go architecture does not currently embed `firebase-admin`, cutting over will result in the temporary loss of the Daily Flash Challenge and the `/storebatch` file indexing search. 

**Recommendation:** Do NOT block production cutover. These features can be re-architected natively into MongoDB `targeted_mongo.go` collections in Phase 15. The core Telegram Bot behaviors (Subscriptions, Support, Requests) are completely decoupled from Firebase.
