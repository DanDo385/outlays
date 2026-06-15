import Link from "next/link";
import { dimensionsFor } from "@/lib/dimensions";

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
      {dimensionsFor(jur).map((d) => (
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
