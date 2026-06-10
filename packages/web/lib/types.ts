// API payload types shared by server pages and client components. View shapes come from
// the contract package — the single source of truth (D26); the remaining shapes mirror
// docs/openapi.yaml, which describes endpoints the contract does not model. Money is
// always a decimal string on the wire — never parse it into a JS number (Hard Rule 2).

import type { FiscalNodeView, FiscalYearView } from "@outlays/contract";

export type Money = string;

export const UNCLASSIFIED = "__unclassified__";

export type ViewNode = FiscalNodeView;
export type View = FiscalYearView;

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
