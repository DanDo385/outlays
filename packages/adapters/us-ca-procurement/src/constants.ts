// Constants and pure transforms for the California procurement adapter.
//
// All upstream query identifiers (resource id, column names) are constants here; no runtime
// input is ever interpolated into a query (Hard Rule 8). The only runtime value used in a
// request is the fiscal year, which is validated against the fiscal-year pattern and mapped
// to the source's format before being passed as a structured CKAN `filters` value (not SQL).

export const JURISDICTION = "us-ca";
export const DATASET = "purchase-order-data";
export const RESOURCE_ID = "bb82edc5-9c78-44e2-8947-68ece26197c5";

export const HOST = "https://data.ca.gov";
export const SEARCH_URL = `${HOST}/api/3/action/datastore_search`;
export const SQL_URL = `${HOST}/api/3/action/datastore_search_sql`;

/** CKAN datastore page size. The adapter pages until a short page (Section: batched ingest). */
export const PAGE_SIZE = 1000;

/** Namespace for provisional vendor entity ids (UUIDv5 of the normalized name). */
export const VENDOR_NAMESPACE = "b6e0a1d2-3c4f-4a5b-8c6d-7e8f9a0b1c2d";

/** Source column names (constants — never built from runtime input). */
export const COL = {
  id: "_id",
  fiscalYear: "Fiscal Year",
  department: "Department Name",
  acquisitionType: "Acquisition Type",
  supplierName: "Supplier Name",
  supplierCode: "Supplier Code",
  totalPrice: "Total Price",
  purchaseDate: "Purchase Date",
  creationDate: "Creation Date",
  itemName: "Item Name",
  poNumber: "Purchase Order Number",
} as const;

/** Source "Fiscal Year" (e.g. "2014-2015") → canonical fiscal-year token ("2014-15"). */
export function toCanonicalFiscalYear(source: string): string | null {
  const range = /^(\d{4})-(\d{4})$/.exec(source.trim());
  if (range) return `${range[1]}-${range[2]!.slice(2)}`;
  if (/^\d{4}$/.test(source.trim())) return source.trim();
  return null;
}

/** Canonical fiscal-year token ("2014-15") → source "Fiscal Year" filter value ("2014-2015"). */
export function toSourceFiscalYear(canonical: string): string | null {
  const m = /^(\d{4})-(\d{2})$/.exec(canonical);
  if (m) {
    const startYear = Number(m[1]);
    const century = Math.floor(startYear / 100) * 100;
    let endYear = century + Number(m[2]);
    if (endYear < startYear) endYear += 100;
    return `${startYear}-${endYear}`;
  }
  if (/^\d{4}$/.test(canonical)) return canonical;
  return null;
}

/** Normalize a money string like "$1,362.00 " to a contract decimal string, or null. */
export function normalizeMoney(raw: string | null | undefined): string | null {
  if (raw == null) return null;
  let s = raw.trim();
  if (!s) return null;
  let negative = false;
  if (/^\(.*\)$/.test(s)) {
    negative = true;
    s = s.slice(1, -1);
  }
  s = s.replace(/[$,\s]/g, "");
  if (s.startsWith("-")) {
    negative = true;
    s = s.slice(1);
  } else if (s.startsWith("+")) {
    s = s.slice(1);
  }
  if (!/^\d+(\.\d+)?$/.test(s)) return null;
  const [intPartRaw, fracRaw = ""] = s.split(".");
  const intPart = intPartRaw!.replace(/^0+(?=\d)/, "");
  const frac = (fracRaw + "0000").slice(0, 4);
  let out = `${intPart}.${frac}`;
  if (negative && !/^0\.0000$/.test(out)) out = `-${out}`;
  return /^-?\d{1,18}(\.\d{1,4})?$/.test(out) ? out : null;
}

/** Normalize a vendor name for provisional matching: trim, collapse whitespace, uppercase.
 *  Deliberately conservative — NO fuzzy merging (no stripping of Inc/LLC/etc.). */
export function normalizeVendorName(raw: string): string {
  return raw.trim().replace(/\s+/g, " ").toUpperCase();
}

/** Treat null / empty / "unknown" supplier names as no identifiable vendor. */
export function isIdentifiableVendor(raw: string | null | undefined): raw is string {
  return !!raw && raw.trim() !== "" && raw.trim().toLowerCase() !== "unknown";
}

/** Parse "M/D/YYYY" or an ISO date into "YYYY-MM-DD", or undefined. */
export function parseDate(raw: string | null | undefined): string | undefined {
  if (!raw) return undefined;
  const s = raw.trim();
  const iso = /^(\d{4}-\d{2}-\d{2})/.exec(s);
  if (iso) return iso[1];
  const mdy = /^(\d{1,2})\/(\d{1,2})\/(\d{4})$/.exec(s);
  if (mdy) {
    const mm = mdy[1]!.padStart(2, "0");
    const dd = mdy[2]!.padStart(2, "0");
    return `${mdy[3]}-${mm}-${dd}`;
  }
  return undefined;
}

/** Provenance string pinning a single source row by its datastore `_id`. */
export function derivationQueryForRow(id: number | string): string {
  return `${SEARCH_URL}?resource_id=${RESOURCE_ID}&filters=${JSON.stringify({ _id: Number(id) })}`;
}
