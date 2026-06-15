/** Compare fiscal years (YYYY or YYYY-YY) for descending sort — newest first. */
export function compareFiscalYears(a: string, b: string): number {
  const start = (y: string) => parseInt(y.slice(0, 4), 10);
  return start(a) - start(b);
}

export function sortFiscalYearsDesc(years: string[]): string[] {
  return [...years].sort((a, b) => compareFiscalYears(b, a));
}

export function latestFiscalYear(years: string[]): string | undefined {
  return sortFiscalYearsDesc(years)[0];
}
