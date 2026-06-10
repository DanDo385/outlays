#!/usr/bin/env node
// California enacted-budget control-total adapter (ebudget.ca.gov).
//
// Emits NO facts — only a ControlTotal per supported year: the official enacted total state
// expenditure figure, with provenance pinned to the fetched document's exact bytes. The
// figure itself is a reviewed in-source constant (see constants.ts); fetching and hashing
// the document at run time makes the transcription independently verifiable. Coverage
// computed against this denominator is labeled by `scope` ("procurement facts vs total
// budget") because the dataset's facts are procurement-only while this total is the full
// state budget.

import { runAdapter, sha256Hex, SourceUnavailableError, type AdapterDefinition } from "@outlays/adapter-sdk";
import { CONTRACT_VERSION, type ControlTotal } from "@outlays/contract";
import { DATASET, JURISDICTION, YEAR_SOURCES } from "./constants.js";

const definition: AdapterDefinition = {
  manifest: {
    adapterId: "us-ca-budget",
    jurisdiction: JURISDICTION,
    datasets: [DATASET],
    adapterVersion: "0.1.0",
    contractVersion: CONTRACT_VERSION,
    license: "Apache-2.0",
    maintainer: "outlays",
  },

  listYears() {
    return Object.keys(YEAR_SOURCES).sort().reverse();
  },

  async fetchYear(ctx) {
    const source = YEAR_SOURCES[ctx.year];
    if (!source) {
      ctx.log("warn", `no enacted-budget source pinned for ${ctx.year}; emitting nothing`);
      return { facts: [] };
    }

    const res = await ctx.fetch(source.url);
    if (res.httpStatus !== 200 || res.bytes.length === 0) {
      throw new SourceUnavailableError(`budget document fetch returned ${res.httpStatus}`);
    }

    const controlTotal: ControlTotal = {
      jurisdiction: JURISDICTION,
      fiscalYear: ctx.year,
      flow: "spending",
      officialTotal: source.officialTotal,
      currency: source.currency,
      scope: source.scope,
      rawSha256: sha256Hex(res.bytes),
      derivationQuery: `${source.locator}; document: ${source.url}`,
    };
    ctx.log("info", `${ctx.year}: control total ${source.officialTotal} ${source.currency} (${source.scope})`);
    return { facts: [], controlTotals: [controlTotal] };
  },
};

void runAdapter(definition);
