import Link from "next/link";
import { getJurisdictions, getYears } from "@/lib/api";
import { jurisdictionName } from "@/lib/dimensions";

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
          The read API is unreachable. Start the stack (<code>docker compose -f
          deploy/docker-compose.yml up -d</code>) and the API (<code>go run ./core/cmd/api</code>,
          with <code>DATABASE_URL</code> set), then reload.
        </div>
      </section>
    );
  }

  const years = await Promise.all(jurisdictions.map((j) => getYears(j)));

  return (
    <section className="landing">
      <h1>Browse the ledger</h1>
      <p className="lede">
        Atomic spending facts, pivotable by department, acquisition type, or payee — the
        same dollars, three lenses. Every figure links to the exact raw bytes it was derived
        from.
      </p>
      <ul className="jur-list">
        {years.map(({ jurisdiction, years: ys }) => (
          <li key={jurisdiction} className="jur-card">
            <h2>{jurisdictionName(jurisdiction)}</h2>
            <div className="years">
              {ys.map((y) => (
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
