#!/usr/bin/env node
"use strict";

// cinedeck npm launcher: locates the platform binary downloaded by install.js
// and spawns it, forwarding argv, stdin/stdout/stderr and the exit code.

const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const BIN_NAME = process.platform === "win32" ? "cinedeck.exe" : "cinedeck";
const binPath = path.join(__dirname, "bin", BIN_NAME);

if (!fs.existsSync(binPath)) {
  console.error(
    "cinedeck: binary not found. Run `npm rebuild cinedeck` or `npm i -g cinedeck` again to download it."
  );
  process.exit(1);
}

const result = spawnSync(binPath, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  console.error("cinedeck: failed to launch binary:", result.error.message);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
