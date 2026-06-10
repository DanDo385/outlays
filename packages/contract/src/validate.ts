// Contract validation against the canonical JSON Schema (draft 2020-12) using ajv.
//
// Validation uses the schema directly (not the generated types) so that conditional rules
// like "transaction/award grain requires rawSha256" are enforced. Generated types are for
// developer ergonomics; the drift guard keeps them in sync with this schema.

import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import _Ajv2020 from "ajv/dist/2020.js";
import _addFormats from "ajv-formats";
import type { ValidateFunction } from "ajv";

// Minimal structural view of the ajv surface we use (avoids naming ajv's namespaced default
// export as a type under NodeNext).
interface AjvLike {
  addSchema(schema: unknown, key?: string): unknown;
  compile(schema: unknown): ValidateFunction;
}

// ajv and ajv-formats are CJS; normalize the interop default under NodeNext and re-type the
// result as a constructor / function (the dual CJS-ESM typings lose the call signature).
const Ajv2020 = ((_Ajv2020 as { default?: unknown }).default ?? _Ajv2020) as {
  new (opts?: Record<string, unknown>): AjvLike;
};
const addFormats = ((_addFormats as { default?: unknown }).default ?? _addFormats) as (ajv: AjvLike) => AjvLike;

/** Absolute path to the canonical schema, resolved relative to this module. */
export const SCHEMA_PATH = fileURLToPath(new URL("../schemas/fiscal.schema.json", import.meta.url));

/** The parsed canonical schema document. */
export const schema = JSON.parse(readFileSync(SCHEMA_PATH, "utf8")) as {
  $id: string;
  $defs: Record<string, unknown>;
};

/** Named top-level contract types (the keys of `$defs`). */
export type ContractDef = string;

const ajv = new Ajv2020({ allErrors: true, strict: false });
addFormats(ajv);
ajv.addSchema(schema, schema.$id);

const validators = new Map<ContractDef, ValidateFunction>();

/** Returns (and memoizes) a compiled validator for a `$defs` type, e.g. "FiscalFact". */
export function validatorFor(def: ContractDef): ValidateFunction {
  const existing = validators.get(def);
  if (existing) return existing;
  if (!(def in schema.$defs)) {
    throw new Error(`unknown contract type '${def}'`);
  }
  const compiled = ajv.compile({ $ref: `${schema.$id}#/$defs/${def}` });
  validators.set(def, compiled);
  return compiled;
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
}

/** Validates `data` against the named contract type. */
export function validate(def: ContractDef, data: unknown): ValidationResult {
  const v = validatorFor(def);
  const valid = v(data) as boolean;
  const errors = valid
    ? []
    : (v.errors ?? []).map((e) => `${e.instancePath || "/"} ${e.message ?? "invalid"}`);
  return { valid, errors };
}
