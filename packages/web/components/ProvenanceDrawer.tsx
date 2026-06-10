"use client";

import { useEffect, useState } from "react";
import type { FactRow, Provenance } from "@/lib/types";
import { formatMoney } from "@/lib/decimal";

/**
 * The citation behind a figure: fact hash, raw-snapshot SHA-256 + object-store key, and
 * the derivation query that selects the exact source row. Fetched through this app's
 * server-side proxy (the browser never talks to the API directly, and never to an LLM).
 */
export function ProvenanceDrawer({ fact, onClose }: { fact: FactRow; onClose: () => void }) {
  const [prov, setProv] = useState<Provenance | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setProv(null);
    setError(null);
    fetch(`/api/provenance/${fact.factId}`)
      .then(async (r) => {
        if (!r.ok) throw new Error(`provenance fetch failed (${r.status})`);
        return (await r.json()) as Provenance;
      })
      .then((p) => {
        if (!cancelled) setProv(p);
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [fact.factId]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <>
      <div className="drawer-backdrop" onClick={onClose} />
      <aside className="drawer" role="dialog" aria-label="Provenance">
        <button className="close" onClick={onClose} aria-label="Close">
          ✕
        </button>
        <h2>Provenance</h2>
        <p className="drawer-sub">
          {formatMoney(fact.amount, fact.currency)} · {fact.grain} grain ·{" "}
          {fact.occurredOn ?? "no date"} — every field below is verifiable against the
          stored raw bytes.
        </p>

        {error && <p className="error">{error}</p>}
        {!prov && !error && <p className="status">Loading provenance…</p>}

        {prov && (
          <dl>
            <dt>Fact ID</dt>
            <dd>{prov.factId}</dd>

            <dt>Fact hash (SHA-256 over RFC 8785 canonical JSON)</dt>
            <dd>{prov.factHash}</dd>

            <dt>Raw snapshot SHA-256 (exact upstream response bytes)</dt>
            <dd>{prov.rawSha256 ?? "—"}</dd>

            <dt>Object-store key (the raw bytes themselves)</dt>
            <dd>{prov.storageKey ?? "—"}</dd>

            <dt>Source URL</dt>
            <dd>{prov.snapshotUrl ?? "—"}</dd>

            <dt>HTTP status / size</dt>
            <dd className="plain">
              {prov.httpStatus ?? "—"} · {prov.bytes != null ? `${prov.bytes} bytes` : "—"}
            </dd>

            <dt>Derivation query (selects the source row)</dt>
            <dd>{prov.derivationQuery}</dd>
          </dl>
        )}

        <p className="verify-note">
          To verify: fetch the object-store key, SHA-256 the bytes (must equal the raw
          snapshot hash), then apply the derivation query to reproduce this row. No step
          requires trusting this UI.
        </p>
      </aside>
    </>
  );
}
