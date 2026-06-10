import type { View } from "@/lib/api";
import { formatMoney, isZero, subMoney } from "@/lib/decimal";
import { Illustrative } from "./Illustrative";

/**
 * Money in − money out over *ingested facts only*. The balance is BigInt minor-units math
 * on the two view totals — no float ever touches an amount.
 */
export function BalanceRibbon({ moneyIn, moneyOut }: { moneyIn: View; moneyOut: View }) {
  const balance = subMoney(moneyIn.total, moneyOut.total);
  const noRevenue = isZero(moneyIn.total) && moneyIn.nodes.length === 0;
  return (
    <section className="ribbon" aria-label="Balance of ingested facts">
      <div>
        <div className="label">
          Money in
          {noRevenue && <Illustrative note="No revenue source has been ingested yet; this side of the ledger is structural only." />}
        </div>
        <div className="value in">{formatMoney(moneyIn.total, moneyIn.currency)}</div>
        <div className="note">
          {noRevenue ? "no revenue source ingested yet" : `${moneyIn.nodes.length} categories`}
        </div>
      </div>
      <div>
        <div className="label">Money out</div>
        <div className="value out">{formatMoney(moneyOut.total, moneyOut.currency)}</div>
        <div className="note">{moneyOut.nodes.length} categories of ingested spending</div>
      </div>
      <div>
        <div className="label">Balance</div>
        <div className="value">{formatMoney(balance, moneyOut.currency)}</div>
        <div className="note">ingested facts only — not an official budget balance</div>
      </div>
    </section>
  );
}
