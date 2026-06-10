import Link from "next/link";
import { notFound } from "next/navigation";
import {
  ApiError,
  UNCLASSIFIED,
  getEntityFlows,
  getNodeFacts,
  getView,
  type EntityFlows,
} from "@/lib/api";
import { dimensionByKey, jurisdictionName, DIMENSIONS } from "@/lib/dimensions";
import { formatMoney } from "@/lib/decimal";
import { FactTable } from "@/components/FactTable";

export const dynamic = "force-dynamic";

const FISCAL_YEAR_RE = /^\d{4}(-\d{2})?$/;
const UUID_RE = /^[0-9a-fA-F-]{36}$/;
const PAGE_SIZE = 100;

export default async function DrillPage({
  params,
  searchParams,
}: {
  params: Promise<{ jur: string; year: string; dim: string; code: string }>;
  searchParams: Promise<{ flow?: string; offset?: string }>;
}) {
  const p = await params;
  const jur = decodeURIComponent(p.jur);
  const year = decodeURIComponent(p.year);
  const code = decodeURIComponent(p.code);
  if (!FISCAL_YEAR_RE.test(year)) notFound();
  if (!DIMENSIONS.some((d) => d.key === p.dim)) notFound();
  const dim = dimensionByKey(p.dim);

  const sp = await searchParams;
  const flow = sp.flow === "revenue" ? "revenue" : "spending";
  const offset = Math.max(0, Number.parseInt(sp.offset ?? "0", 10) || 0);

  let view, page;
  try {
    // The view supplies the node's label and exact total; the facts call is the page of
    // underlying PO-level rows for that node.
    [view, page] = await Promise.all([
      getView(jur, year, dim.scheme, flow),
      getNodeFacts(jur, year, flow, dim.scheme, code, PAGE_SIZE, offset),
    ]);
  } catch (e) {
    if (e instanceof ApiError && e.status === 400) notFound();
    throw e;
  }
  const node = view.nodes.find((n) => n.code === code);
  if (!node) notFound();

  // Payee drill-down: also show this vendor across departments (the D1 cross-cut).
  let crosscut: EntityFlows | null = null;
  if (dim.key === "payee" && code !== UNCLASSIFIED && UUID_RE.test(code)) {
    crosscut = await getEntityFlows(code, year);
  }

  const hasPrev = offset > 0;
  const hasNext = offset + page.facts.length < node.factCount;
  const pageHref = (o: number) =>
    `/${jur}/${year}/drill/${dim.key}/${encodeURIComponent(code)}?flow=${flow}&offset=${o}`;

  return (
    <>
      <nav className="crumbs">
        <Link href={`/${jur}/${year}?dim=${dim.key}`}>
          {jurisdictionName(jur)} FY {year}
        </Link>{" "}
        / {dim.label.toLowerCase()} / {node.label}
      </nav>

      <div className="drill-head">
        <h1>{node.label}</h1>
        <span className="amount" title={`${node.amount} ${node.currency} (exact)`}>
          {formatMoney(node.amount, node.currency)}
        </span>
      </div>
      <p className="drill-sub">
        {node.factCount} {flow} fact{node.factCount === 1 ? "" : "s"} at award grain (one
        per purchase-order line). Click a row for its provenance — the hashes and query
        that tie the figure to the raw source bytes.
      </p>

      {crosscut && crosscut.byDepartment.length > 0 && (
        <section className="crosscut">
          <h2>Same payee across departments</h2>
          <ul>
            {crosscut.byDepartment.map((d) => (
              <li key={d.code}>
                {d.label}{" "}
                <span className="amount">{formatMoney(d.amount, d.currency)}</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      <FactTable facts={page.facts} />

      {(hasPrev || hasNext) && (
        <nav className="pager">
          {hasPrev ? (
            <Link href={pageHref(Math.max(0, offset - PAGE_SIZE))}>← Previous</Link>
          ) : (
            <span className="disabled">← Previous</span>
          )}
          <span className="disabled">
            {offset + 1}–{offset + page.facts.length} of {node.factCount}
          </span>
          {hasNext ? (
            <Link href={pageHref(offset + PAGE_SIZE)}>Next →</Link>
          ) : (
            <span className="disabled">Next →</span>
          )}
        </nav>
      )}
    </>
  );
}
