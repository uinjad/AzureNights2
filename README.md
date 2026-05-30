# AzureNights2

A turn-based terminal RPG with an L2/Ragnarok-style class advancement tree,
built in Go with a clean hexagonal architecture.

## Status
Work in progress — building in the open, one focused commit at a time.

## Run
```bash
make run
```

## Architecture
Pure domain core, adapters on the outside (TUI now; storage and an optional
HTTP server are plug-in adapters). See `internal/domain` for the engine.