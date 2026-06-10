import Link from "next/link";
import { UNCLASSIFIED, type View } from "@/lib/api";
import { formatMoney, sharePercent } from "@/lib/decimal";
import { Illustrative } from "./Illustrative";

/**
 * One side of the two-sided ledger: a one-level rollup of the requested scheme. Every node
 * (including the explicit unclassified bucket — never hidden) drills down to its facts.
 */
export function LedgerSide({
  title,
  side,
  view,
  dimKey,
}: {
  title: string;
  side: "in" | "out";
  view: View;
  dimKey: string;
}) {
  const empty = view.nodes.length === 0;
  return (
    <section className={`ledger-side side-${side}`}>
      <header>
        <h2>{title}</h2>
        <span className="side-total">{formatMoney(view.total, view.currency)}</span>
      </header>

      {empty ? (
        <div className="empty-side">
          <Illustrative note="No facts ingested for this flow; the rows below sketch the layout only and carry no figures." />
          <p>
            No {view.flow} facts have been ingested for {view.jurisdiction} FY{" "}
            {view.fiscalYear}. When a {view.flow} source lands, its categories will appear
            here exactly like the other side — the sketch below is layout only, with no
            figures.
          </p>
          <ul className="sketch" aria-hidden="true">
            {["", "", ""].map((_, i) => (
              <li key={i}>
                <span>· · · · · · · · · · · ·</span>
                <span className="dash">—</span>
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <ul className="node-list">
          {view.nodes.map((n) => (
            <li key={n.code}>
              <Link
                className="node-row"
                href={`/${view.jurisdiction}/${view.fiscalYear}/drill/${dimKey}/${encodeURIComponent(n.code)}?flow=${view.flow}`}
              >
                <span
                  className="bar"
                  style={{ width: `${sharePercent(n.amount, view.total)}%` }}
                />
                <span className="line">
                  <span className={n.code === UNCLASSIFIED ? "name unclassified" : "name"}>
                    {n.label}
                  </span>
                  <span className="count">
                    {n.factCount} fact{n.factCount === 1 ? "" : "s"}
                  </span>
                  <span className="amount" title={`${n.amount} ${n.currency} (exact)`}>
                    {formatMoney(n.amount, n.currency)}
                  </span>
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}

      <div className="side-footnote">
        {empty
          ? "Nothing here is invented: zero is the honest total of ingested facts."
          : `Unmapped (no assignment in this scheme): ${formatMoney(view.unmapped, view.currency)} — shown above as “Unclassified”, never dropped.`}
      </div>
    </section>
  );
}
