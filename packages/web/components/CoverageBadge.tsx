import type { Coverage } from "@/lib/api";
import { formatMoney, percentFromRatio } from "@/lib/decimal";

/**
 * Coverage = ingested transaction+award facts / official control total. Until a
 * control_total is ingested (S8) the denominator is null — rendered honestly as
 * "coverage unknown", never as 100%.
 */
export function CoverageBadge({ coverage }: { coverage: Coverage }) {
  if (coverage.denominator === null || coverage.ratio === null) {
    return (
      <span
        className="badge badge-coverage-unknown"
        title={`Ingested facts total ${formatMoney(coverage.numerator, coverage.currency)}; no official control total ingested yet, so the share of total spending is unknown.`}
      >
        coverage unknown — no official total ingested yet
      </span>
    );
  }
  return (
    <span
      className="badge badge-coverage-known"
      title={`${formatMoney(coverage.numerator, coverage.currency)} of ${formatMoney(coverage.denominator, coverage.currency)} official total`}
    >
      coverage {percentFromRatio(coverage.ratio)} of official total
    </span>
  );
}
