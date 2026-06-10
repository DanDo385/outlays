// Package store owns persistence: Postgres (system of record) and S3-compatible object
// storage for raw snapshots. Implemented in S4 (goose migrations, append-only enforcement,
// batched COPY writer, object-store writer). S0: package marker only.
package store

// Version identifies the store package surface; bumped as the schema evolves.
const Version = "0.0.0"
