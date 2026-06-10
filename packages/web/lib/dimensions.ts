// The three pivot dimensions over the same facts (D1). Schemes are the CA per-source
// schemes for now (Phase 0 is California end to end); when a second jurisdiction lands,
// this becomes a per-jurisdiction lookup.

export interface Dimension {
  key: string;
  label: string;
  scheme: string;
}

export const DIMENSIONS: readonly Dimension[] = [
  { key: "department", label: "By department", scheme: "us_ca_department" },
  { key: "acquisition", label: "By acquisition type", scheme: "us_ca_acquisition_type" },
  { key: "payee", label: "By payee", scheme: "payee" },
] as const;

export const DEFAULT_DIMENSION = DIMENSIONS[0]!;

export function dimensionByKey(key: string | undefined): Dimension {
  return DIMENSIONS.find((d) => d.key === key) ?? DEFAULT_DIMENSION;
}

/** Display names for jurisdiction codes (fallback: the code itself). */
const JURISDICTION_NAMES: Record<string, string> = {
  "us-ca": "California",
};

export function jurisdictionName(jur: string): string {
  return JURISDICTION_NAMES[jur] ?? jur;
}
