"use client";

import { useRouter } from "next/navigation";
import { sortFiscalYearsDesc } from "@/lib/fiscalYear";

export function YearSwitcher({
  jur,
  year,
  years,
  dim,
}: {
  jur: string;
  year: string;
  years: string[];
  dim: string;
}) {
  const router = useRouter();
  const ordered = sortFiscalYearsDesc(years);
  return (
    <label className="year-switcher">
      Fiscal year
      <select
        value={year}
        onChange={(e) => router.push(`/${jur}/${e.target.value}?dim=${dim}`)}
      >
        {ordered.map((y) => (
          <option key={y} value={y}>
            {y}
          </option>
        ))}
      </select>
    </label>
  );
}
