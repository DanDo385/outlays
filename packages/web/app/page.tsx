import { redirect } from "next/navigation";
import Link from "next/link";
import { getJurisdictions, getYears } from "@/lib/api";
import { jurisdictionName } from "@/lib/dimensions";
import { latestFiscalYear, sortFiscalYearsDesc } from "@/lib/fiscalYear";

export const dynamic = "force-dynamic";

export default async function Landing() {
  let jurisdictions: string[];
  try {
    jurisdictions = (await getJurisdictions()).jurisdictions;
  } catch {
    return (
      <section className="landing">
        <h1>Outlays</h1>
        <p className="lede">
          Make any government-spending question answerable in seconds, with a verifiable
          citation back to the source row.
        </p>
        <div className="error-panel">
          The read API is unreachable. Start the stack (<code>make up</code>), run the API (
          <code>make run-api</code>), then reload.
        </div>
      </section>
    );
  }

  if (jurisdictions.length === 0) {
    return (
      <section className="landing">
        <h1>Browse the ledger</h1>
        <p className="lede">No jurisdictions ingested yet. Run <code>make seed</code> to load
        the federal FY 2025 demo.</p>
      </section>
    );
  }

  const entries = await Promise.all(
    jurisdictions.map(async (j) => ({ jurisdiction: j, years: (await getYears(j)).years })),
  );
  entries.sort((a, b) => a.jurisdiction.localeCompare(b.jurisdiction));

  const primary = entries[0]!;
  const latest = latestFiscalYear(primary.years);
  if (latest) {
    redirect(`/${primary.jurisdiction}/${latest}`);
  }

  return (
    <section className="landing">
      <h1>Browse the ledger</h1>
      <p className="lede">
        Atomic spending facts, pivotable by department, acquisition type, or payee — the
        same dollars, three lenses. Every figure links to the exact raw bytes it was derived
        from.
      </p>
      <ul className="jur-list">
        {entries.map(({ jurisdiction, years }) => (
          <li key={jurisdiction} className="jur-card">
            <h2>{jurisdictionName(jurisdiction)}</h2>
            <div className="years">
              {sortFiscalYearsDesc(years).map((y) => (
                <Link key={y} href={`/${jurisdiction}/${y}`}>
                  FY {y}
                </Link>
              ))}
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}
