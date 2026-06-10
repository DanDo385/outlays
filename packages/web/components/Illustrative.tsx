/** Marker for anything on screen that is not backed by live-ingested facts. */
export function Illustrative({ note }: { note?: string }) {
  return (
    <span className="badge badge-illustrative" title={note ?? "Not live-ingested data."}>
      illustrative — not live data
    </span>
  );
}
