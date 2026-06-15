import type { Metadata } from "next";
import Link from "next/link";
import "./globals.css";

export const metadata: Metadata = {
  title: "Outlays",
  description:
    "Government spending, answerable in seconds, with a verifiable citation back to the source row.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    // Extensions (e.g. SwiftRead) mutate <html>/<body> before hydration; suppress the noise.
    <html lang="en" suppressHydrationWarning>
      <body suppressHydrationWarning>
        <header className="site-header">
          <Link href="/" className="brand">
            Outlays
          </Link>
          <span className="tagline">
            Every figure traces to source bytes. Neutral method, no editorializing.
          </span>
        </header>
        <main className="site-main">{children}</main>
        <footer className="site-footer">
          Read-only over the public Outlays API · money is exact decimal, never floated ·
          unmapped is shown, never hidden
        </footer>
      </body>
    </html>
  );
}
