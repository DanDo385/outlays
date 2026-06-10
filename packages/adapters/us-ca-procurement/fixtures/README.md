# us-ca-procurement fixtures

Recorded from one live run against data.ca.gov on 2026-06-09 (CKAN resource
`bb82edc5-9c78-44e2-8947-68ece26197c5`, "Purchase Order Data"). CI replays these and never
hits the network (Hard Rule 9).

`replay/` holds the HTTP record/replay cache used by the SDK when `OUTLAYS_REPLAY_DIR` points
at it:

- `index.json` — maps each request URL to its recorded body file + HTTP status.
- `<hash>.bin` — exact upstream response bytes (one for the `list-years` distinct-years SQL
  query, one for the first 1000-row page of FY 2014-15).

**Replaying requires `OUTLAYS_MAX_PAGES=1`** so the adapter requests exactly the one recorded
page (the live dataset has ~116k rows/year across ~116 pages; only the first is recorded).
With that env set, `fetch --year 2014-15` is fully offline and reproducible:

```sh
OUTLAYS_REPLAY_DIR=packages/adapters/us-ca-procurement/fixtures/replay \
OUTLAYS_MAX_PAGES=1 \
node packages/adapters/us-ca-procurement/dist/cli.js fetch --year 2014-15 --raw-dir /tmp/raw --out /tmp/out.json
```

To re-record (live network), set `OUTLAYS_RECORD_DIR` to the `replay/` dir and run the same
`list-years` and `fetch` commands.
