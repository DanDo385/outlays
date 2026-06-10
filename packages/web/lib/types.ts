// API payload types shared by server pages and client components. Shapes mirror
// docs/openapi.yaml (the served read-API payloads). Money is always a decimal string on
// the wire — never parse it into a JS number (Hard Rule 2).

export type Money = string;

export const UNCLASSIFIED = "__unclassified__";

export interface ViewNode {
  code: string;
  label: string;
  amount: Money;
  currency: string;
  factCount: number;
}

export interface View {
  jurisdiction: string;
  fiscalYear: string;
  flow: "spending" | "revenue";
  schemeId: string;
  total: Money;
  currency: string;
  unmapped: Money;
  nodes: ViewNode[];
}

export interface Coverage {
  jurisdiction: string;
  fiscalYear: string;
  numerator: Money;
  denominator: Money | null;
  ratio: string | null;
  currency: string;
}

export interface FactRow {
  factId: string;
  jurisdiction: string;
  fiscalYear: string;
  flow: string;
  grain: string;
  amount: Money;
  currency: string;
  occurredOn: string | null;
  description: string | null;
  payeeEntity: string | null;
  factHash: string;
}

export interface Provenance {
  factId: string;
  factHash: string;
  derivationQuery: string;
  rawSha256: string | null;
  storageKey: string | null;
  snapshotUrl: string | null;
  httpStatus: number | null;
  bytes: number | null;
}

export interface EntityFlows {
  entityId: string;
  canonicalName: string;
  fiscalYear: string;
  total: Money;
  currency: string;
  byDepartment: ViewNode[];
}
