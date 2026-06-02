# Contributing

Thanks for your interest in AzureNights.

## Development

```bash
make test    # run the full suite under the race detector
make ci      # what CI enforces: gofmt, vet, test, build
make run     # play it
make balance # headless balance report
```

## Architecture rules

The dependency arrow points inward (see the README diagram):

- `internal/domain/*` is pure — no I/O, no time, no randomness. Inject those.
- `internal/content` validates JSON into typed registries at load.
- `internal/app` holds use-cases and the `Repository` port; adapters depend on it.
- `internal/tui` and `internal/storage` are adapters. `cmd/*` wires everything.

New content (classes, enemies, maps, quests) should be JSON under
`internal/content/data/` — adding a zone shouldn't require touching Go.

## Pull requests

- Keep the domain test-covered and deterministic.
- One logical change per PR, with a conventional-commit title.
- `make ci` must pass.