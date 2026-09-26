# FINAL CUTOVER BLOCKERS

## 1. MONGODB `ReplaceOne` OVERWRITE RISK (P0 - BLOCKER)
**Component**: `internal/database/mongo.go` (Line 166)
**Impact**: When Go's `MongoStore.Save(ctx, state)` is called (which occurs on startup during token/auth routines and periodically during runtime), it executes a `ReplaceOne` operation. 
**Why this is a Blocker**: `ReplaceOne` completely replaces the existing `main_settings` document in MongoDB with the `BotState` Go struct. Any legacy Python fields (e.g., deprecated variables, unused fields, or unmapped structures) that are NOT explicitly modeled in Go's `BotState` will be **permanently wiped out**. This violates the absolute rule of non-destructive migration and prevents a safe rollback to Python if those fields were needed by Python.
**Required Fix**: The `Save()` routine MUST be rewritten to utilize `$set` exclusively (similar to how `targeted_mongo.go` operates) rather than completely overwriting the document payload.

## 2. FIREBASE MISSING INTEGRATION (P2 - Non-Critical)
**Component**: `app.py` Gamification / `handlers.py` Storebatch
**Impact**: Users interacting with Flashcards or `/storebatch` indexes will not receive expected behaviors since Go lacks the `firebase-admin` package mapping.
**Why this is NOT a blocker**: These features are disconnected from core Bot functionality (Authentication, Batch Access, Support Topics). They are cosmetic/ancillary and can be reimplemented in Phase 17 securely.

## 3. UNKNOWN / UNDOCUMENTED LEGACY FIELDS (P1)
**Component**: Python `DB` Dictionary -> Go `BotState`
**Impact**: While most fields have been mapped perfectly, if Python was actively mutating undocumented keys inside `main_settings.data`, Go ignores them on read. Combined with the P0 `ReplaceOne` bug above, these would be lost forever.
**Mitigation**: Solving Blocker #1 (`ReplaceOne` -> `$set`) entirely eliminates this risk because Go will simply ignore unknown fields instead of destroying them.
