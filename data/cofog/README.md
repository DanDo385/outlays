# COFOG mapping files

Reviewed mapping files consumed by the S9 classification loader. One file per source
scheme; format:

```json
{
  "<sourceCategory>": { "cofogCode": "07", "confidence": 0.9, "basis": "<citation/reasoning>" }
}
```

**Builders never invent mappings.** These files are produced by research and land via PRs.
Until a real file exists, the loader is exercised against a fixture and real facts are left
unmapped — unmapped is honest (Hard Rule 5).
