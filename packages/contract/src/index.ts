// @outlays/contract — single source of truth for cross-language types.
//
// Types in ./generated are produced from schemas/fiscal.schema.json by codegen and kept in
// sync by the CI drift guard. Runtime validation lives in ./validate (ajv against the same
// schema, so conditional rules are enforced).

export * from "./generated/types.js";
export {
  validate,
  validatorFor,
  schema,
  SCHEMA_PATH,
  type ContractDef,
  type ValidationResult,
} from "./validate.js";

/** Envelope version pinned by the contract. */
export const ENVELOPE_VERSION = "1" as const;

/** Contract version reported by adapter manifests (`contractVersion`). */
export const CONTRACT_VERSION = "0.3.0" as const;

/** Decimal-string money regex (mirrors ARCHITECTURE.md Hard Rule 2). */
export const MONEY_PATTERN = /^-?\d{1,18}(\.\d{1,4})?$/;

/** Fiscal-year param regex (mirrors Hard Rule 8). */
export const FISCAL_YEAR_PATTERN = /^\d{4}(-\d{2})?$/;
