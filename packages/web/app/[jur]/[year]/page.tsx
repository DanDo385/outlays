import { notFound } from "next/navigation";
import { ApiError, getCoverage, getView, getYears } from "@/lib/api";
import { dimensionByKey, jurisdictionName } from "@/lib/dimensions";
import { BalanceRibbon } from "@/components/BalanceRibbon";
import { CoverageBadge } from "@/components/CoverageBadge";
import { DimensionSwitcher } from "@/components/DimensionSwitcher";
import { LedgerSide } from "@/components/LedgerSide";
import { YearSwitcher } from "@/components/YearSwitcher";

export const dynamic = "force-dynamic";

const FISCAL_YEAR_RE = /^\d{4}(-\d{2})?$/;

export default async function LedgerPage({
  params,
  searchParams,
}: {
  params: Promise<{ jur: string; year: string }>;
  searchParams: Promise<{ dim?: string }>;
}) {
  const { jur: rawJur, year: rawYear } = await params;
  const jur = decodeURIComponent(rawJur);
  const year = decodeURIComponent(rawYear);
  if (!FISCAL_YEAR_RE.test(year)) notFound();
  const dim = dimensionByKey((await searchParams).dim);

  let spending, revenue, coverage, years;
  try {
    // The same facts, both flows of the requested dimension, in parallel.
    [spending, revenue, coverage, years] = await Promise.all([
      getView(jur, year, dim.scheme, "spending"),
      getView(jur, year, dim.scheme, "revenue"),
      getCoverage(jur, year),
      getYears(jur),
    ]);
  } catch (e) {
    if (e instanceof ApiError && e.status === 400) notFound();
    throw e;
  }

  return (
    <>
      <div className="masthead">
        <h1>
          {jurisdictionName(jur)} <span style={{ fontWeight: 400 }}>FY {year}</span>
        </h1>
        <CoverageBadge coverage={coverage} />
        <span className="spacer" />
        <YearSwitcher jur={jur} year={year} years={years.years} dim={dim.key} />
      </div>

      <DimensionSwitcher jur={jur} year={year} active={dim.key} />

      <BalanceRibbon moneyIn={revenue} moneyOut={spending} />

      <div className="ledger">
        <LedgerSide title="Money in" side="in" view={revenue} dimKey={dim.key} />
        <LedgerSide title="Money out" side="out" view={spending} dimKey={dim.key} />
      </div>
    </>
  );
}
