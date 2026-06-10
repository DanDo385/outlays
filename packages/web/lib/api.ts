// Server-side client for the Outlays read API (docs/openapi.yaml). All fetches happen in
// server components / route handlers — the browser only ever talks to this Next app, and
// no LLM is called anywhere (Section 5: no LLM calls from the web client).
//
// Payload shapes live in lib/types.ts: view types come from the contract package (the
// single source of truth, D26); the rest mirror docs/openapi.yaml, which describes the
// endpoints the contract does not model.

import "server-only";

import type { Coverage, EntityFlows, FactRow, Provenance, View } from "./types";

export type { Coverage, EntityFlows, FactRow, Money, Provenance, View, ViewNode } from "./types";
export { UNCLASSIFIED } from "./types";

const API_BASE = process.env.OUTLAYS_API_URL ?? "http://localhost:8080";

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly path: string,
    message: string,
  ) {
    super(`GET ${path} -> ${status}: ${message}`);
  }
}

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, { cache: "no-store" });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new ApiError(res.status, path, body.slice(0, 200));
  }
  return (await res.json()) as T;
}

export function getJurisdictions(): Promise<{ jurisdictions: string[] }> {
  return getJSON(`/v1/jurisdictions`);
}

export function getYears(jur: string): Promise<{ jurisdiction: string; years: string[] }> {
  return getJSON(`/v1/${encodeURIComponent(jur)}/years`);
}

export function getView(
  jur: string,
  year: string,
  scheme: string,
  flow: "spending" | "revenue",
): Promise<View> {
  return getJSON(
    `/v1/${encodeURIComponent(jur)}/${encodeURIComponent(year)}/view?scheme=${encodeURIComponent(scheme)}&flow=${flow}`,
  );
}

export function getCoverage(jur: string, year: string): Promise<Coverage> {
  return getJSON(`/v1/${encodeURIComponent(jur)}/${encodeURIComponent(year)}/coverage`);
}

export function getNodeFacts(
  jur: string,
  year: string,
  flow: "spending" | "revenue",
  scheme: string,
  code: string,
  limit: number,
  offset: number,
): Promise<{ limit: number; offset: number; facts: FactRow[] }> {
  const p = new URLSearchParams({
    jurisdiction: jur,
    year,
    flow,
    scheme,
    code,
    limit: String(limit),
    offset: String(offset),
  });
  return getJSON(`/v1/facts?${p.toString()}`);
}

export function getProvenance(factId: string): Promise<Provenance> {
  return getJSON(`/v1/fact/${encodeURIComponent(factId)}/provenance`);
}

export function getEntityFlows(entityId: string, year: string): Promise<EntityFlows> {
  return getJSON(`/v1/entities/${encodeURIComponent(entityId)}/flows?year=${encodeURIComponent(year)}`);
}
