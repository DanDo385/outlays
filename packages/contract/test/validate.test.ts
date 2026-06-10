// Runs the shared fixture cases (fixtures/cases.json) through the ajv validators and asserts
// each fixture's actual validity matches its declared `valid` flag. Exits non-zero on any
// mismatch so it can gate CI. The same cases run in Python and Go; all three must agree.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { validate } from "../src/validate.js";

interface Case {
  file: string;
  def: string;
  valid: boolean;
}

const casesPath = fileURLToPath(new URL("../fixtures/cases.json", import.meta.url));
const fixturesRoot = fileURLToPath(new URL("../fixtures/", import.meta.url));
const { cases } = JSON.parse(readFileSync(casesPath, "utf8")) as { cases: Case[] };

let failures = 0;
for (const c of cases) {
  const data = JSON.parse(readFileSync(new URL(c.file, `file://${fixturesRoot}`), "utf8"));
  const result = validate(c.def, data);
  const ok = result.valid === c.valid;
  const status = ok ? "PASS" : "FAIL";
  const detail = result.valid ? "(accepted)" : `(rejected: ${result.errors.join("; ")})`;
  console.log(`${status}  ${c.def.padEnd(24)} expect=${c.valid ? "valid " : "invalid"} ${c.file} ${detail}`);
  if (!ok) failures++;
}

if (failures > 0) {
  console.error(`\n${failures} fixture(s) did not match expected validity`);
  process.exit(1);
}
console.log(`\nAll ${cases.length} contract fixtures matched expected validity.`);
