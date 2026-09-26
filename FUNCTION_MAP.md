# FUNCTION MAP

## Core Routing
Python `bot.py` (@app.on_message) -> Go `internal/bot/router.go` (HandleUpdate)

## Handlers
Python `handlers.py:start_command` -> Go `internal/bot/handlers_start.go:HandleStart`
Python `handlers.py:id_command` -> Go `internal/bot/handlers_start.go:HandleID`
Python `handlers.py:admin_command` -> Go `internal/bot/handlers_start.go:HandleAdmin`

## Services
Python `handlers.py` (Topic creation logic) -> Go `internal/services/support.go:EnsureTopic`
Python `handlers.py` (Broadcast logic) -> Go `internal/services/broadcast.go:RunBroadcast`

## Excluded (Web Only)
Python `app.py` -> Excluded
Python `html_generator.py` -> Excluded
Python `templates/` -> Excluded
Python `static/` -> Excluded

## Missing (Needs further MTProto migration)
Python `handlers.py` (Userbot History scanning) -> Requires `github.com/gotd/td` client integration (Future work)
