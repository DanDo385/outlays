#!/usr/bin/env node
// California procurement adapter — data.ca.gov CKAN "Purchase Order Data"
// (resource bb82edc5-9c78-44e2-8947-68ece26197c5).
//
// Emits one award-grain fact per source row (a purchase-order line item): amount = Total
// Price, with provenance pinned to the row's datastore _id and the page snapshot bytes.
// Department and acquisition type become source-coded classification assignments; supplier
// names become provisional vendor entities + aliases (UUIDv5 of the normalized name, exact
// match only — NO fuzzy merging). Designed for batched paging over 300k+ rows.

import {
  httpGet,
  runAdapter,
  sha256Hex,
  uuidv5,
  SourceUnavailableError,
  type AdapterDefinition,
  type FactInput,
} from "@outlays/adapter-sdk";
import { CONTRACT_VERSION, type Entity, type EntityAlias } from "@outlays/contract";
import {
  COL,
  DATASET,
  JURISDICTION,
  PAGE_SIZE,
  RESOURCE_ID,
  SEARCH_URL,
  SQL_URL,
  VENDOR_NAMESPACE,
  derivationQueryForRow,
  isIdentifiableVendor,
  normalizeMoney,
  normalizeVendorName,
  parseDate,
  toCanonicalFiscalYear,
  toSourceFiscalYear,
} from "./constants.js";

type Row = Record<string, string | null | undefined>;

interface CkanResponse {
  success: boolean;
  result?: { records?: Row[]; total?: number };
  error?: unknown;
}

function parseCkan(bytes: Uint8Array): CkanResponse {
  return JSON.parse(Buffer.from(bytes).toString("utf8")) as CkanResponse;
}

function maxPages(): number {
  const env = process.env["OUTLAYS_MAX_PAGES"];
  const n = env ? Number(env) : NaN;
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : Number.POSITIVE_INFINITY;
}

const definition: AdapterDefinition = {
  manifest: {
    adapterId: "us-ca-procurement",
    jurisdiction: JURISDICTION,
    datasets: [DATASET],
    adapterVersion: "0.1.0",
    contractVersion: CONTRACT_VERSION,
    license: "Apache-2.0",
    maintainer: "outlays",
  },

  async listYears() {
    const sql = `SELECT DISTINCT "${COL.fiscalYear}" FROM "${RESOURCE_ID}"`;
    const url = `${SQL_URL}?${new URLSearchParams({ sql }).toString()}`;
    const res = await httpGet(url);
    const json = parseCkan(res.bytes);
    if (!json.success || !json.result?.records) {
      throw new SourceUnavailableError("listYears: datastore_search_sql failed");
    }
    const canonical = new Set<string>();
    for (const r of json.result.records) {
      const c = toCanonicalFiscalYear(String(r[COL.fiscalYear] ?? ""));
      if (c) canonical.add(c);
    }
    return [...canonical].sort().reverse();
  },

  async fetchYear(ctx) {
    const sourceYear = toSourceFiscalYear(ctx.year);
    if (!sourceYear) {
      ctx.log("warn", `no source fiscal year mapping for ${ctx.year}`);
      return { facts: [] };
    }

    const facts: FactInput[] = [];
    const entities = new Map<string, Entity>();
    const aliases = new Map<string, EntityAlias>();
    let skippedNoAmount = 0;
    let total: number | undefined;

    const limit = maxPages();
    for (let page = 0, offset = 0; page < limit; page++, offset += PAGE_SIZE) {
      const params = new URLSearchParams({
        resource_id: RESOURCE_ID,
        limit: String(PAGE_SIZE),
        offset: String(offset),
        filters: JSON.stringify({ [COL.fiscalYear]: sourceYear }),
      });
      const url = `${SEARCH_URL}?${params.toString()}`;
      const res = await ctx.fetch(url);
      const json = parseCkan(res.bytes);
      if (!json.success) {
        throw new SourceUnavailableError(`fetch page ${page}: CKAN returned success=false`);
      }
      const records = json.result?.records ?? [];
      total = json.result?.total ?? total;
      const pageSha = sha256Hex(res.bytes);

      for (const r of records) {
        const amount = normalizeMoney(r[COL.totalPrice]);
        if (!amount) {
          skippedNoAmount++;
          continue;
        }

        const assignments: NonNullable<FactInput["assignments"]> = [];
        const dept = r[COL.department]?.trim();
        if (dept) {
          assignments.push({
            schemeId: "us_ca_department",
            code: dept,
            assignedBy: "source",
            version: 1,
            basis: "Department Name as coded by data.ca.gov Purchase Order Data",
          });
        }
        const acq = r[COL.acquisitionType]?.trim();
        if (acq) {
          assignments.push({
            schemeId: "us_ca_acquisition_type",
            code: acq,
            assignedBy: "source",
            version: 1,
            basis: "Acquisition Type as coded by data.ca.gov Purchase Order Data",
          });
        }

        let payeeEntity: string | undefined;
        const rawSupplier = r[COL.supplierName];
        if (isIdentifiableVendor(rawSupplier)) {
          const canonicalName = normalizeVendorName(rawSupplier);
          const entityId = uuidv5(`${JURISDICTION}:vendor:${canonicalName}`, VENDOR_NAMESPACE);
          payeeEntity = entityId;
          if (!entities.has(entityId)) {
            entities.set(entityId, { entityId, kind: "vendor", canonicalName, jurisdiction: JURISDICTION });
          }
          const aliasKey = `${entityId}|${rawSupplier}`;
          if (!aliases.has(aliasKey)) {
            aliases.set(aliasKey, {
              entityId,
              nameRaw: rawSupplier,
              matchedBy: "rule",
              confidence: 0.5,
              source: {
                dataset: DATASET,
                supplierCode: r[COL.supplierCode] ?? null,
                normalized: canonicalName,
              },
            });
          }
        }

        const occurredOn = parseDate(r[COL.purchaseDate]) ?? parseDate(r[COL.creationDate]);
        const description = r[COL.itemName]?.trim() || undefined;

        const fact: FactInput = {
          jurisdiction: JURISDICTION,
          fiscalYear: ctx.year,
          flow: "spending",
          grain: "award",
          amount,
          currency: "USD",
          rawSha256: pageSha,
          derivationQuery: derivationQueryForRow(r[COL.id] ?? ""),
          ...(payeeEntity ? { payeeEntity } : {}),
          ...(occurredOn ? { occurredOn } : {}),
          ...(description ? { description } : {}),
          ...(assignments.length ? { assignments } : {}),
        };
        facts.push(fact);
      }

      ctx.log("info", `page ${page}: ${records.length} rows (offset ${offset}, total ${total ?? "?"})`);
      if (records.length < PAGE_SIZE) break;
    }

    ctx.log(
      "info",
      `${ctx.year}: ${facts.length} facts, ${entities.size} vendors, ${aliases.size} aliases, ${skippedNoAmount} rows skipped (no amount)`,
    );
    return { facts, entities: [...entities.values()], entityAliases: [...aliases.values()] };
  },
};

void runAdapter(definition);
