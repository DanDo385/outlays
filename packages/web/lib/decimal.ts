// Money handling for the web client. Hard Rule 2: amounts are decimal strings end to end;
// any arithmetic happens in BigInt minor units (1/10000). JS `number` never holds an amount.

const MONEY_RE = /^-?\d{1,18}(\.\d{1,4})?$/;

/** Parse a contract decimal string into BigInt minor units (4 decimal places). */
export function toMinor(amount: string): bigint {
  if (!MONEY_RE.test(amount)) {
    throw new Error(`not a contract money string: ${JSON.stringify(amount)}`);
  }
  const neg = amount.startsWith("-");
  const body = neg ? amount.slice(1) : amount;
  const dot = body.indexOf(".");
  const intPart = dot === -1 ? body : body.slice(0, dot);
  const frac = (dot === -1 ? "" : body.slice(dot + 1)).padEnd(4, "0");
  const n = BigInt(intPart) * 10000n + BigInt(frac);
  return neg ? -n : n;
}

/** Format BigInt minor units back to a contract decimal string (4 decimal places). */
export function fromMinor(minor: bigint): string {
  const neg = minor < 0n;
  const abs = neg ? -minor : minor;
  const intPart = (abs / 10000n).toString();
  const frac = (abs % 10000n).toString().padStart(4, "0");
  return `${neg ? "-" : ""}${intPart}.${frac}`;
}

export function addMoney(a: string, b: string): string {
  return fromMinor(toMinor(a) + toMinor(b));
}

export function subMoney(a: string, b: string): string {
  return fromMinor(toMinor(a) - toMinor(b));
}

export function isNegative(amount: string): boolean {
  return toMinor(amount) < 0n;
}

export function isZero(amount: string): boolean {
  return toMinor(amount) === 0n;
}

/**
 * Human display of a decimal-string amount: thousands separators, 2 decimal places
 * (4 when the sub-cent digits are nonzero — never silently rounded). Pure string work.
 */
export function formatMoney(amount: string, currency: string): string {
  const neg = amount.startsWith("-");
  const body = neg ? amount.slice(1) : amount;
  const dot = body.indexOf(".");
  const intPart = dot === -1 ? body : body.slice(0, dot);
  const frac = (dot === -1 ? "" : body.slice(dot + 1)).padEnd(4, "0");
  const grouped = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  const shownFrac = frac.slice(2) === "00" ? frac.slice(0, 2) : frac;
  const symbol = currency === "USD" ? "$" : `${currency} `;
  return `${neg ? "−" : ""}${symbol}${grouped}.${shownFrac}`;
}

/**
 * Share of `part` in `total` as a percentage for bar widths — a display ratio, not money,
 * so returning a JS number is fine. Computed in BigInt basis points first.
 */
export function sharePercent(part: string, total: string): number {
  const t = toMinor(total);
  if (t <= 0n) return 0;
  const bp = (toMinor(part) * 10000n) / t;
  const pct = Number(bp) / 100;
  return Math.max(0, Math.min(100, pct));
}

/** Render an API ratio string (e.g. "0.000247") as a percentage without floats. */
export function percentFromRatio(ratio: string): string {
  if (!/^\d+(\.\d+)?$/.test(ratio)) return ratio;
  const dot = ratio.indexOf(".");
  const intPart = dot === -1 ? ratio : ratio.slice(0, dot);
  const frac = (dot === -1 ? "" : ratio.slice(dot + 1)).padEnd(6, "0").slice(0, 6);
  const ratioMicro = BigInt(intPart) * 1000000n + BigInt(frac);
  const pctMicro = ratioMicro * 100n;
  const whole = pctMicro / 1000000n;
  const frac6 = (pctMicro % 1000000n).toString().padStart(6, "0");
  const trimmed = `${whole}.${frac6}`.replace(/0+$/, "").replace(/\.$/, "");
  return `${trimmed}%`;
}

/** True when ratio > 1.0 (ingested facts exceed the official control total). */
export function ratioExceedsOne(ratio: string): boolean {
  if (!/^\d+(\.\d+)?$/.test(ratio)) return false;
  const dot = ratio.indexOf(".");
  const intPart = dot === -1 ? ratio : ratio.slice(0, dot);
  if (BigInt(intPart) > 1n) return true;
  if (BigInt(intPart) < 1n) return false;
  const frac = (dot === -1 ? "" : ratio.slice(dot + 1)).replace(/0+$/, "");
  return frac.length > 0;
}
