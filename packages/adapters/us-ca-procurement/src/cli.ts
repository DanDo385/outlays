#!/usr/bin/env node
// California procurement adapter (data.ca.gov CKAN "Purchase Order Data").
//
// Implemented for real in S3. For S0 this is a buildable placeholder exposing the adapter
// CLI shape so the workspace compiles.

import { ExitCode } from "@outlays/adapter-sdk";

function main(argv: string[]): number {
  const [cmd] = argv;
  switch (cmd) {
    case "info":
    case "list-years":
    case "fetch":
      process.stderr.write(
        JSON.stringify({ level: "error", msg: `command '${cmd}' not implemented until S3`, ts: new Date().toISOString() }) + "\n",
      );
      return ExitCode.Unexpected;
    default:
      process.stderr.write(
        JSON.stringify({ level: "error", msg: `unknown command '${cmd ?? ""}'`, ts: new Date().toISOString() }) + "\n",
      );
      return ExitCode.Unexpected;
  }
}

process.exit(main(process.argv.slice(2)));
