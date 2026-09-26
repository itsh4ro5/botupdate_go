# PHASE 17: FIREBASE FINAL DECISION

## 1. Firebase Usage in Python (`app.py` & `handlers.py`)
Firebase Firestore is integrated strictly for two operational subsets:
1. **Gamification**: Users answering flashcard questions via `/api/flash/generate` and `/api/flash/current` save their scores to Firestore.
2. **StoreBatch Search**: PDFs/Videos uploaded to designated storage channels have their Chat ID indexed inside Firestore for the `/storebatch` command lookup.

## 2. Dependency Audit
| Feature | Python Usage | Data Stored | Read By Python? | Written By Go? | Status | Decision |
| ------- | ------------ | ----------- | --------------- | -------------- | ------ | -------- |
| Flashcards | Extensively in `app.py` (API) | Quiz scores/prompts | YES | NO | Non-blocking | **DEPRECATE** |
| StoreBatch | `handlers.py:3023` | Chat IDs | YES | NO | Non-blocking | **DEPRECATE** |

## 3. Justification for Deprecation
Neither of these collections intersect with the Bot's Authentication (Tokens, User Registration, Blocked Users) or Core Business Logic (Batch Purchases, Access Validation, Support Channels). 
Removing Firebase entirely from the Go stack poses **ZERO** risk of cross-pollinating or destroying existing MongoDB state. The Python implementation handles its own Firebase credentials without MongoDB dependencies.

**Final Decision**: **DEPRECATE**. The cutover can proceed safely. If Gamification or StoreBatch search is desired post-cutover, they will be built natively into MongoDB via `targeted_mongo.go` inside Phase 18.
