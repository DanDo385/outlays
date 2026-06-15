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
export type SchemeId =
  | "cofog"
  | "object_class"
  | "department"
  | "fund"
  | "program"
  | "recipient_type"
  | "tag"
  | "us_ca_department"
  | "us_ca_acquisition_type"
  | "us_fed_awarding_agency"
  | "us_fed_award_type";

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
  AdapterOutput?: AdapterOutput;
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
/**
 * An official published total for a jurisdiction-year-flow, used as the coverage denominator. scope is the explicit label for what coverage against this total means (e.g. 'procurement facts vs total budget') — it must name any scope mismatch between the ingested facts and this denominator so coverage never implies false precision.
 */
export interface ControlTotal {
  jurisdiction: string;
  fiscalYear: FiscalYear;
  flow: Flow;
  officialTotal: Money;
  currency: Iso4217;
  scope: string;
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
 * The document an adapter writes to --out: the run envelope plus the derived facts (and any provisional entities/aliases). envelope.resultHash is the SHA-256 over RFC 8785 (JCS) canonical JSON of the facts, sorted by factHash — deterministic across runs.
 */
export interface AdapterOutput {
  envelope: IngestionEnvelope;
  facts: FiscalFact[];
  entities?: Entity[];
  entityAliases?: EntityAlias[];
  controlTotals?: ControlTotal[];
}
/**
 * One row of a computed view: a scheme code (or entity id, or __unclassified__) with its rolled-up amount (API payload only, NOT storage). The scheme is carried once on the enclosing FiscalYearView.
 */
export interface FiscalNodeView {
  code: string;
  label: string;
  amount: Money;
  currency: Iso4217;
  factCount: number;
}
/**
 * A one-level jurisdiction-year rollup over one scheme/flow (API payload only, NOT storage). unmapped is the total of facts with no assignment in the scheme, also surfaced as the __unclassified__ node — node amounts always reconcile to total (D24).
 */
export interface FiscalYearView {
  jurisdiction: string;
  fiscalYear: FiscalYear;
  flow: Flow;
  schemeId: string;
  total: Money;
  currency: Iso4217;
  unmapped: Money;
  nodes: FiscalNodeView[];
}
