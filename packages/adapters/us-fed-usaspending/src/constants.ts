export const JURISDICTION = "us-fed";
export const DATASET = "usaspending-assistance";
export const API_URL = "https://api.usaspending.gov/api/v2/search/spending_by_award/";
export const PAGE_SIZE = 100;

/** Federal assistance award type codes (grants, cooperative agreements, etc.). */
export const ASSISTANCE_AWARD_TYPES = ["02", "03", "04", "05"] as const;

export const FIELDS = [
  "Award ID",
  "Recipient Name",
  "Recipient UEI",
  "Award Amount",
  "Awarding Agency",
  "Award Type",
  "generated_internal_id",
] as const;

/** Federal fiscal year token (e.g. "2025") to USAspending obligation period. */
export function fiscalYearPeriod(year: string): { startDate: string; endDate: string } | undefined {
  const m = /^(\d{4})$/.exec(year);
  if (!m) return undefined;
  const y = Number(m[1]);
  return { startDate: `${y - 1}-10-01`, endDate: `${y}-09-30` };
}

export const ENTITY_NAMESPACE = "6ba7b810-9dad-11d1-80b4-00c04fd430c8";

export function normalizeMoney(amount: unknown): string | undefined {
  if (typeof amount === "number" && Number.isFinite(amount)) {
    const cents = Math.round(amount * 10000);
    const neg = cents < 0;
    const abs = Math.abs(cents);
    const whole = Math.floor(abs / 10000);
    const frac = (abs % 10000).toString().padStart(4, "0");
    return `${neg ? "-" : ""}${whole}.${frac}`;
  }
  if (typeof amount === "string" && /^-?\d+(\.\d{1,4})?$/.test(amount.trim())) {
    const t = amount.trim();
    const dot = t.indexOf(".");
    if (dot === -1) return `${t}.0000`;
    const frac = t.slice(dot + 1).padEnd(4, "0").slice(0, 4);
    return `${t.slice(0, dot)}.${frac}`;
  }
  return undefined;
}

export function normalizeRecipientName(raw: string): string {
  return raw.trim().replace(/\s+/g, " ");
}

export function derivationQueryForAward(awardId: string, internalId: string): string {
  return `USAspending spending_by_award; Award ID=${awardId}; internal_id=${internalId}`;
}
