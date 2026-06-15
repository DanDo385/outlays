// Pivot dimensions per jurisdiction (D1). Schemes must exist in classification_scheme.

export interface Dimension {
  key: string;
  label: string;
  scheme: string;
}

const CA_DIMENSIONS: readonly Dimension[] = [
  { key: "department", label: "By department", scheme: "us_ca_department" },
  { key: "acquisition", label: "By acquisition type", scheme: "us_ca_acquisition_type" },
  { key: "payee", label: "By payee", scheme: "payee" },
];

const FED_DIMENSIONS: readonly Dimension[] = [
  { key: "agency", label: "By awarding agency", scheme: "us_fed_awarding_agency" },
  { key: "award_type", label: "By award type", scheme: "us_fed_award_type" },
  { key: "payee", label: "By recipient", scheme: "payee" },
];

const DIMENSIONS_BY_JUR: Record<string, readonly Dimension[]> = {
  "us-ca": CA_DIMENSIONS,
  "us-fed": FED_DIMENSIONS,
};

export function dimensionsFor(jur: string): readonly Dimension[] {
  return DIMENSIONS_BY_JUR[jur] ?? CA_DIMENSIONS;
}

export function dimensionByKey(jur: string, key: string | undefined): Dimension {
  const dims = dimensionsFor(jur);
  return dims.find((d) => d.key === key) ?? dims[0]!;
}

/** Display names for jurisdiction codes (fallback: the code itself). */
const JURISDICTION_NAMES: Record<string, string> = {
  "us-ca": "California",
  "us-fed": "United States (federal)",
};

export function jurisdictionName(jur: string): string {
  return JURISDICTION_NAMES[jur] ?? jur;
}
