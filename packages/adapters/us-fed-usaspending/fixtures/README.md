# USAspending replay fixtures (federal FY2025 assistance demo)

Recorded responses for `OUTLAYS_REPLAY_DIR`. CI and `make seed` use these only; no live
government API calls in CI.

## Record (maintainers)

```sh
pnpm --filter @outlays/adapter-us-fed-usaspending build
OUTLAYS_RECORD_DIR=packages/adapters/us-fed-usaspending/fixtures/replay \
  OUTLAYS_MAX_PAGES=1 \
  node packages/adapters/us-fed-usaspending/dist/cli.js fetch --year 2025 --raw-dir /tmp/raw --out /tmp/out.json
```

Commit the updated `fixtures/replay/index.json` and `*.bin` files.
