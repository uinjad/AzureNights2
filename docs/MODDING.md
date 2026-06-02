# Modding AzureNights

All content lives in `internal/content/data/` as JSON and is embedded into the
binary at build time. Edit the JSON, run `go build ./cmd/rpg`, and your changes
ship. Everything is validated at load — a broken reference fails fast at startup
with a clear message, never mid-game.

## Add a class
Edit `data/classes.json`. A class is a tree node with attribute bonuses, skills,
and advancement edges:

    {"id":"solar_paladin","name":"Solar Paladin","faction":"solar",
     "bonus":{"str":6,"con":7,"men":3},"skills":["radiant_smite"]}

Then point an existing class's `advances` at it. Skill ids must exist in
`skills.json`; advancement targets must exist as classes.

## Add a faction
Edit `data/factions.json`. Factions form a cycle via `beats`; the multipliers
tune how hard the triangle bites. Re-balance, then run `go run ./cmd/balance`
to see the new win rates through the real combat engine.

## Add an enemy
Edit `data/enemies.json`: stats, faction, and rewards. Place it on a map.

## Add or edit a map
Drop a `data/maps/<name>.json`. Rows reference a `legend`; `enemies`, `portals`,
and `rests` are coordinate lists. A portal's `to_map` must name another map.

## Add a quest
Edit `data/quests.json`. Objective kinds:
- `defeat` — `target` is an enemy id, `count` is how many.
- `reach`  — `target` is a map id.

Targets are checked against the enemies and maps you've defined.

## Verify
`make test` proves the engine still holds; `make balance` reports class and
faction balance; `make run` plays it.