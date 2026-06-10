import type { Coverage } from "@/lib/api";
import { formatMoney, percentFromRatio } from "@/lib/decimal";

/**
 * Coverage = ingested transaction+award facts / official control total. With no control
 * total ingested the denominator is null — rendered honestly as "coverage unknown", never
 * as 100%. With one, the badge always carries the control total's scope label (e.g.
 * "procurement facts vs total budget") so a scope mismatch between the facts and the
 * denominator is stated, never implied away.
 */
export function CoverageBadge({ coverage }: { coverage: Coverage }) {
  if (coverage.denominator === null || coverage.ratio === null || coverage.denominatorBasis === null) {
    return (
      <span
        className="badge badge-coverage-unknown"
        title={`Ingested facts total ${formatMoney(coverage.numerator, coverage.currency)}; no official control total ingested yet, so the share of total spending is unknown.`}
      >
        coverage unknown — no official total ingested yet
      </span>
    );
  }
  const basis = coverage.denominatorBasis;
  return (
    <span
      className="badge badge-coverage-known"
      title={`${formatMoney(coverage.numerator, coverage.currency)} ingested facts of ${formatMoney(coverage.denominator, coverage.currency)} official total. Denominator: ${basis.derivationQuery}`}
    >
      coverage {percentFromRatio(coverage.ratio)} — {basis.scope}
    </span>
  );
}
