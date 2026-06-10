#!/usr/bin/env node
// Toy fixture adapter: a deterministic, network-free adapter used to exercise the SDK and the
// conformance harness. It synthesizes a small raw dataset, snapshots the exact bytes, and
// emits one award-grain fact per row. Same input ⇒ same resultHash, every run.
//
// The raw payload and derivation are byte-for-byte identical to the Python toy adapter
// (py/adapter_sdk/examples/toy_fixture.py), so both produce the same resultHash — a
// cross-language check on JCS hashing.

import { runAdapter, type AdapterDefinition, type FactInput } from "@outlays/adapter-sdk";
import { CONTRACT_VERSION, type Entity, type EntityAlias } from "@outlays/contract";

// Exact bytes of the synthetic upstream response (ASCII, no trailing newline).
const RAW =
  '[{"po":"PO-1","dept":"DGS","vendor":"Acme Supplies Inc","amount":"1000.0000","date":"2024-07-01"},' +
  '{"po":"PO-2","dept":"CDCR","vendor":"Beacon Logistics LLC","amount":"25000.5000","date":"2024-08-15"},' +
  '{"po":"PO-3","dept":"CDCR","vendor":"Acme Supplies Inc","amount":"750.2500","date":"2024-09-30"}]';

interface Row {
  po: string;
  dept: string;
  vendor: string;
  amount: string;
  date: string;
}

const definition: AdapterDefinition = {
  manifest: {
    adapterId: "toy-fixture",
    jurisdiction: "us-xx",
    datasets: ["toy"],
    adapterVersion: "0.1.0",
    contractVersion: CONTRACT_VERSION,
    license: "Apache-2.0",
    maintainer: "outlays",
  },

  listYears() {
    return ["2024-25", "2023-24"];
  },

  async fetchYear(ctx) {
    const bytes = Buffer.from(RAW, "utf8");
    const snap = await ctx.snapshot({ url: `toy://toy/${ctx.year}`, bytes, httpStatus: 200 });
    const rows = JSON.parse(RAW) as Row[];

    const facts: FactInput[] = rows.map((r) => ({
      jurisdiction: "us-xx",
      fiscalYear: ctx.year,
      flow: "spending",
      grain: "award",
      amount: r.amount,
      currency: "USD",
      occurredOn: r.date,
      description: r.vendor,
      rawSha256: snap.sha256,
      derivationQuery: `toy:row:${r.po}`,
      assignments: [
        {
          schemeId: "department",
          code: r.dept,
          assignedBy: "source",
          version: 1,
          basis: "toy: department as coded by source",
        },
      ],
    }));

    const vendorNames = [...new Set(rows.map((r) => r.vendor))];
    const entities: Entity[] = vendorNames.map((name) => ({
      kind: "vendor",
      canonicalName: name,
      jurisdiction: "us-xx",
    }));
    const entityAliases: EntityAlias[] = vendorNames.map((name) => ({
      nameRaw: name,
      matchedBy: "rule",
      source: { normalized: name.toLowerCase() },
    }));

    ctx.log("info", `toy-fixture produced ${facts.length} facts for ${ctx.year}`);
    return { facts, entities, entityAliases };
  },
};

void runAdapter(definition);
