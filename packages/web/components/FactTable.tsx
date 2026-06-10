"use client";

import { useState } from "react";
import type { FactRow } from "@/lib/types";
import { formatMoney } from "@/lib/decimal";
import { ProvenanceDrawer } from "./ProvenanceDrawer";

/** PO-level fact rows; clicking one opens the provenance drawer for that exact fact. */
export function FactTable({ facts }: { facts: FactRow[] }) {
  const [selected, setSelected] = useState<FactRow | null>(null);

  return (
    <>
      <table className="fact-table">
        <thead>
          <tr>
            <th>Date</th>
            <th>Description</th>
            <th>Grain</th>
            <th className="amount">Amount</th>
            <th aria-label="Provenance" />
          </tr>
        </thead>
        <tbody>
          {facts.map((f) => (
            <tr key={f.factId} onClick={() => setSelected(f)} title="Show provenance">
              <td className="date">{f.occurredOn ?? "—"}</td>
              <td className="desc">{f.description ?? "(no description in source)"}</td>
              <td>
                <span className="badge badge-grain">{f.grain}</span>
              </td>
              <td className="amount" title={`${f.amount} ${f.currency} (exact)`}>
                {formatMoney(f.amount, f.currency)}
              </td>
              <td className="prov-hint">provenance ›</td>
            </tr>
          ))}
        </tbody>
      </table>

      {selected && (
        <ProvenanceDrawer fact={selected} onClose={() => setSelected(null)} />
      )}
    </>
  );
}
