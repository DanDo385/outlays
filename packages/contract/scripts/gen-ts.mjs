// Generates TypeScript types for every `$defs` entry in the canonical schema.
// json-schema-to-typescript only emits types reachable from the root, so we feed it a
// wrapper whose properties reference each $def. Output is deterministic (drift-guarded).

import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { compile } from "json-schema-to-typescript";

const root = new URL("../", import.meta.url);
const schemaPath = fileURLToPath(new URL("schemas/fiscal.schema.json", root));
const outPath = fileURLToPath(new URL("src/generated/types.ts", root));

const schema = JSON.parse(readFileSync(schemaPath, "utf8"));

const wrapper = {
  $schema: schema.$schema,
  $id: schema.$id,
  $defs: schema.$defs,
  title: "OutlaysContract",
  type: "object",
  additionalProperties: false,
  properties: Object.fromEntries(
    Object.keys(schema.$defs).map((name) => [name, { $ref: `#/$defs/${name}` }]),
  ),
};

const banner = `// GENERATED FILE — DO NOT EDIT.
// Source of truth: packages/contract/schemas/fiscal.schema.json
// Regenerate: pnpm --filter @outlays/contract codegen`;

const ts = await compile(wrapper, "OutlaysContract", {
  bannerComment: banner,
  additionalProperties: false,
  declareExternallyReferenced: true,
  enableConstEnums: false,
  format: true,
});

mkdirSync(fileURLToPath(new URL("src/generated/", root)), { recursive: true });
writeFileSync(outPath, ts);
console.log(`wrote ${outPath}`);
