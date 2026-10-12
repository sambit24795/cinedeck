"use strict";

// Downloads the prebuilt cinedeck binary for the current platform from the
// matching GitHub release, based on the npm package version.

const fs = require("node:fs");
const path = require("node:path");
const zlib = require("node:zlib");
const { Readable } = require("node:stream");
const { pipeline } = require("node:stream/promises");
const tar = require("tar");

const OWNER = "sambit24795";
const REPO = "cinedeck";
const BIN_NAME = process.platform === "win32" ? "cinedeck.exe" : "cinedeck";

function platformTag() {
  const os = process.platform; // linux, darwin, win32, android
  const arch = process.arch; // x64, arm64, ia32, ...
  // GoReleaser names the Windows target "windows", Node calls it "win32".
  const goOS = os === "win32" ? "windows" : os;
  const goArch = arch === "x64" ? "amd64" : arch === "arm64" ? "arm64" : null;
  if (!goArch) {
    throw new Error(`cinedeck: unsupported architecture ${arch}`);
  }
  return { os: goOS, goArch };
}

function assetName(version, os, goArch) {
  return `cinedeck_${version}_${os}_${goArch}.tar.gz`;
}

function version() {
  const pkg = JSON.parse(
    fs.readFileSync(path.join(__dirname, "package.json"), "utf8")
  );
  return pkg.version;
}

async function main() {
  const v = version();
  const { os, goArch } = platformTag();

  // Windows archives still contain the `.exe` binary under its own name.
  const binDir = path.join(__dirname, "bin");
  fs.mkdirSync(binDir, { recursive: true });

  const url = `https://github.com/${OWNER}/${REPO}/releases/download/v${v}/${assetName(
    v,
    os,
    goArch
  )}`;

  console.error(`cinedeck: downloading ${url}`);

  const res = await fetch(url, { redirect: "follow" });
  if (!res.ok) {
    throw new Error(
      `cinedeck: download failed (HTTP ${res.status}); is release v${v} published for ${os}/${goArch}?`
    );
  }

  // The archive contains a single "cinedeck(.exe)" file; extract just it.
  await pipeline(
    Readable.fromWeb(res.body),
    zlib.createGunzip(),
    tar.extract({ cwd: binDir, strip: 0, filter: () => true })
  );

  const bin = path.join(binDir, BIN_NAME);
  if (!fs.existsSync(bin)) {
    throw new Error(
      `cinedeck: archive did not contain ${BIN_NAME}; please report this.`
    );
  }
  fs.chmodSync(bin, 0o755);

  console.error(`cinedeck: installed ${bin}`);
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
