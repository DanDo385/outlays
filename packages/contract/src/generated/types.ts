// GENERATED FILE — DO NOT EDIT.
// Source of truth: packages/contract/schemas/fiscal.schema.json
// Regenerate: pnpm --filter @outlays/contract codegen

/**
 * Monetary amount as a decimal string. JS number / floats forbidden (Hard Rule 2).
 */
export type Money = string;
/**
 * ISO 4217 currency code.
 */
export type Iso4217 = string;
/**
 * Fiscal year token (Hard Rule 8): YYYY or YYYY-YY.
 */
export type FiscalYear = string;
/**
 * Lowercase hex SHA-256 digest.
 */
export type Sha256 = string;
export type Uuid = string;
export type Flow = "revenue" | "spending";
export type Grain = "transaction" | "award" | "aggregate";
export type EntityKind = "government" | "vendor" | "nonprofit" | "individual" | "unknown";
export type AssignedBy = "source" | "rule" | "model" | "human";
/**
 * Known classification schemes. Closed set: an unknown scheme is rejected (mirrors the DB FK to classification_scheme). New per-source schemes are added here as the contract evolves.
 */
export type SchemeId = "cofog" | "object_class" | "department" | "fund" | "program" | "recipient_type" | "tag";

export interface OutlaysContract {
  Money?: Money;
  Iso4217?: Iso4217;
  FiscalYear?: FiscalYear;
  Sha256?: Sha256;
  Uuid?: Uuid;
  Flow?: Flow;
  Grain?: Grain;
  EntityKind?: EntityKind;
  AssignedBy?: AssignedBy;
  SchemeId?: SchemeId;
  SourceRef?: SourceRef;
  Entity?: Entity;
  EntityAlias?: EntityAlias;
  ClassificationAssignment?: ClassificationAssignment;
  FiscalFact?: FiscalFact;
  ControlTotal?: ControlTotal;
  RawSnapshotRef?: RawSnapshotRef;
  IngestionEnvelope?: IngestionEnvelope;
  FiscalNodeView?: FiscalNodeView;
  FiscalYearView?: FiscalYearView;
}
export interface SourceRef {
  jurisdiction: string;
  dataset: string;
  resourceId: string;
  derivationQuery: string;
  pulledAt: string;
  rawSha256: Sha256;
}
export interface Entity {
  entityId?: Uuid;
  kind: EntityKind;
  canonicalName: string;
  uei?: string;
  ein?: string;
  jurisdiction?: string;
}
export interface EntityAlias {
  aliasId?: Uuid;
  entityId?: Uuid;
  nameRaw: string;
  matchedBy: "identifier" | "rule" | "model" | "human";
  confidence?: number;
  source: {};
}
export interface ClassificationAssignment {
  assignmentId?: Uuid;
  factId?: Uuid;
  schemeId: SchemeId;
  code: string;
  assignedBy: AssignedBy;
  confidence?: number;
  basis?: string;
  version: number;
}
/**
 * The atom. Mirrors fiscal_fact. Provenance required (Hard Rule 1): derivationQuery always; rawSha256 for transaction/award grain. amount is a decimal string (Hard Rule 2).
 */
export interface FiscalFact {
  factId?: Uuid;
  runId?: Uuid;
  jurisdiction: string;
  fiscalYear: FiscalYear;
  flow: Flow;
  grain: Grain;
  payerEntity?: Uuid;
  payeeEntity?: Uuid;
  amount: Money;
  currency: Iso4217;
  occurredOn?: string;
  description?: string;
  rawSha256?: Sha256;
  derivationQuery: string;
  factHash: Sha256;
  supersedes?: Uuid;
  assignments?: ClassificationAssignment[];
}
export interface ControlTotal {
  jurisdiction: string;
  fiscalYear: FiscalYear;
  flow: Flow;
  officialTotal: Money;
  rawSha256: Sha256;
  derivationQuery: string;
}
export interface RawSnapshotRef {
  sha256: Sha256;
  url: string;
  bytes: number;
  httpStatus: number;
}
/**
 * Result of one adapter run. signature/signerKeyId are present now (null) so Phase-2 federated signed submissions need no schema rework (Decision D6).
 */
export interface IngestionEnvelope {
  envelopeVersion: "1";
  adapterId: string;
  adapterVersion: string;
  runId: Uuid;
  fetchedAt: string;
  rawSnapshots: RawSnapshotRef[];
  jurisdiction: string;
  fiscalYear: FiscalYear;
  resultHash: Sha256;
  signature: string | null;
  signerKeyId: string | null;
}
/**
 * One level of a computed category tree (API payload only, NOT storage).
 */
export interface FiscalNodeView {
  schemeId: string;
  code: string;
  label: string;
  amount: Money;
  currency: Iso4217;
  factCount: number;
  hasChildren?: boolean;
}
/**
 * A jurisdiction-year view over one scheme/flow (API payload only, NOT storage).
 */
export interface FiscalYearView {
  jurisdiction: string;
  fiscalYear: FiscalYear;
  flow: Flow;
  schemeId: string;
  path?: string[];
  total: Money;
  currency: Iso4217;
  unmapped?: Money;
  nodes: FiscalNodeView[];
}
