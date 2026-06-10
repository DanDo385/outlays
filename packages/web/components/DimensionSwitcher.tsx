import Link from "next/link";
import { DIMENSIONS } from "@/lib/dimensions";

/** Tabs that re-pivot the same facts over a different scheme via the view endpoint (D1). */
export function DimensionSwitcher({
  jur,
  year,
  active,
}: {
  jur: string;
  year: string;
  active: string;
}) {
  return (
    <nav className="dim-switcher" aria-label="Pivot dimension">
      {DIMENSIONS.map((d) => (
        <Link
          key={d.key}
          href={`/${jur}/${year}?dim=${d.key}`}
          className={d.key === active ? "active" : undefined}
        >
          {d.label}
        </Link>
      ))}
    </nav>
  );
}
