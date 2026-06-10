// Server-side proxy for the provenance drawer: the browser stays same-origin and the read
// API never needs CORS. Pass-through only — no reshaping of sourced figures.

import { NextResponse } from "next/server";

const API_BASE = process.env.OUTLAYS_API_URL ?? "http://localhost:8080";
const UUID_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;

export async function GET(
  _req: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  const { id } = await params;
  if (!UUID_RE.test(id)) {
    return NextResponse.json({ error: "invalid fact id" }, { status: 400 });
  }
  const res = await fetch(`${API_BASE}/v1/fact/${id}/provenance`, { cache: "no-store" });
  const body = await res.json().catch(() => ({ error: "bad upstream response" }));
  return NextResponse.json(body, { status: res.status });
}
