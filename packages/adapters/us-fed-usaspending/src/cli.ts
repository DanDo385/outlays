#!/usr/bin/env node
// Federal USAspending assistance adapter (demo path).
//
// Emits one award-grain fact per assistance award from the public spending_by_award search
// API for a federal fiscal year (e.g. 2025 = Oct 2024 through Sep 2025). Assignments:
// us_fed_awarding_agency, us_fed_award_type. Recipients with a UEI become entities with
// kind nonprofit when the name suggests a nongovernment organization, else vendor.

import {
  runAdapter,
  sha256Hex,
  uuidv5,
  SourceUnavailableError,
  type AdapterDefinition,
  type FactInput,
} from "@outlays/adapter-sdk";
import { CONTRACT_VERSION, type Entity, type EntityAlias } from "@outlays/contract";
import {
  API_URL,
  ASSISTANCE_AWARD_TYPES,
  DATASET,
  ENTITY_NAMESPACE,
  FIELDS,
  JURISDICTION,
  PAGE_SIZE,
  derivationQueryForAward,
  fiscalYearPeriod,
  normalizeMoney,
  normalizeRecipientName,
} from "./constants.js";

interface SearchResponse {
  results?: Record<string, unknown>[];
  page_metadata?: { hasNext?: boolean };
}

function maxPages(): number {
  const env = process.env["OUTLAYS_MAX_PAGES"];
  const n = env ? Number(env) : NaN;
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : Number.POSITIVE_INFINITY;
}

function entityKind(name: string): Entity["kind"] {
  const u = name.toUpperCase();
  if (
    u.includes("DEPARTMENT OF") ||
    u.includes("DEPT OF") ||
    u.startsWith("STATE OF ") ||
    u.includes(" COUNTY OF ") ||
    u.endsWith(" COUNTY") ||
    u.includes(" CITY OF ")
  ) {
    return "government";
  }
  return "nonprofit";
}

function searchBody(year: string, page: number): string {
  const period = fiscalYearPeriod(year);
  if (!period) throw new SourceUnavailableError(`unsupported fiscal year ${year}`);
  return JSON.stringify({
    filters: {
      time_period: [{ start_date: period.startDate, end_date: period.endDate }],
      award_type_codes: [...ASSISTANCE_AWARD_TYPES],
    },
    fields: [...FIELDS],
    limit: PAGE_SIZE,
    page,
    sort: "Award Amount",
    order: "desc",
  });
}

const definition: AdapterDefinition = {
  manifest: {
    adapterId: "us-fed-usaspending",
    jurisdiction: JURISDICTION,
    datasets: [DATASET],
    adapterVersion: "0.1.0",
    contractVersion: CONTRACT_VERSION,
    license: "Apache-2.0",
    maintainer: "outlays",
  },

  listYears() {
    return ["2025", "2024"];
  },

  async fetchYear(ctx) {
    if (!fiscalYearPeriod(ctx.year)) {
      ctx.log("warn", `no USAspending period for ${ctx.year}`);
      return { facts: [] };
    }

    const facts: FactInput[] = [];
    const entities = new Map<string, Entity>();
    const aliases = new Map<string, EntityAlias>();
    let skippedNoAmount = 0;

    const limit = maxPages();
    for (let page = 1; page <= limit; page++) {
      const body = searchBody(ctx.year, page);
      const res = await ctx.fetch(API_URL, { body });
      const json = JSON.parse(Buffer.from(res.bytes).toString("utf8")) as SearchResponse;
      if (!json.results) {
        throw new SourceUnavailableError(`fetch page ${page}: missing results`);
      }
      const pageSha = sha256Hex(res.bytes);

      for (const row of json.results) {
        const amount = normalizeMoney(row["Award Amount"]);
        if (!amount) {
          skippedNoAmount++;
          continue;
        }
        const awardId = String(row["Award ID"] ?? "").trim();
        const internalId = String(row["generated_internal_id"] ?? row["internal_id"] ?? awardId);
        if (!awardId) continue;

        const assignments: NonNullable<FactInput["assignments"]> = [];
        const agency = String(row["Awarding Agency"] ?? "").trim();
        if (agency) {
          assignments.push({
            schemeId: "us_fed_awarding_agency",
            code: agency,
            assignedBy: "source",
            version: 1,
            basis: "Awarding Agency as coded by USAspending spending_by_award",
          });
        }
        const awardType = String(row["Award Type"] ?? "").trim();
        if (awardType) {
          assignments.push({
            schemeId: "us_fed_award_type",
            code: awardType,
            assignedBy: "source",
            version: 1,
            basis: "Award Type as coded by USAspending spending_by_award",
          });
        }

        let payeeEntity: string | undefined;
        const rawName = String(row["Recipient Name"] ?? "").trim();
        const uei = String(row["Recipient UEI"] ?? "").trim();
        if (rawName && uei) {
          const canonicalName = normalizeRecipientName(rawName);
          const entityId = uuidv5(`${JURISDICTION}:uei:${uei}`, ENTITY_NAMESPACE);
          payeeEntity = entityId;
          if (!entities.has(entityId)) {
            entities.set(entityId, {
              entityId,
              kind: entityKind(canonicalName),
              canonicalName,
              jurisdiction: JURISDICTION,
              uei,
            });
          }
          const aliasKey = `${entityId}|${rawName}`;
          if (!aliases.has(aliasKey)) {
            aliases.set(aliasKey, {
              entityId,
              nameRaw: rawName,
              matchedBy: "identifier",
              confidence: 1,
              source: { dataset: DATASET, uei },
            });
          }
        }

        facts.push({
          jurisdiction: JURISDICTION,
          fiscalYear: ctx.year,
          flow: "spending",
          grain: "award",
          amount,
          currency: "USD",
          rawSha256: pageSha,
          derivationQuery: derivationQueryForAward(awardId, internalId),
          ...(payeeEntity ? { payeeEntity } : {}),
          description: rawName || undefined,
          ...(assignments.length ? { assignments } : {}),
        });
      }

      ctx.log("info", `page ${page}: ${json.results.length} awards`);
      if (!json.page_metadata?.hasNext || json.results.length < PAGE_SIZE) break;
    }

    ctx.log(
      "info",
      `${ctx.year}: ${facts.length} facts, ${entities.size} recipients, ${skippedNoAmount} skipped (no amount)`,
    );
    return { facts, entities: [...entities.values()], entityAliases: [...aliases.values()] };
  },
};

void runAdapter(definition);
