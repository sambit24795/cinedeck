"use strict";

// Downloads the prebuilt cinedeck binary for the current platform from the
// matching GitHub release, based on the npm package version.

const fs = require("node:fs");
const path = require("node:path");
const zlib = require("node:zlib");
const { Readable } = require("node:stream");
const { pipeline } = require("node:stream/promises");
const tar = require("tar");
const { ensureInstalledOnPath, isGlobalInstall } = require("./winpath");

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

  const bin = placeBinary(binDir, BIN_NAME);
  if (!bin) {
    throw new Error(
      `cinedeck: archive did not contain ${BIN_NAME}; please report this.`
    );
  }
  try {
    fs.chmodSync(bin, 0o755);
  } catch (err) {
    // Windows only honors the read-only bit; a failure here must not
    // roll back an otherwise successful install.
    if (process.platform !== "win32") throw err;
  }

  console.error(`cinedeck: installed ${bin}`);
  registerOnPath(path.dirname(bin));
}

function placeBinary(binDir, name) {
  const direct = path.join(binDir, name);
  if (fs.existsSync(direct)) return direct;

  const found = findBinary(binDir, name);
  if (!found) return null;
  if (path.resolve(found) === path.resolve(direct)) return direct;
  fs.renameSync(found, direct);
  return direct;
}

function findBinary(dir, name) {
  const wanted = name.toLowerCase();
  const stack = [dir];
  while (stack.length) {
    const current = stack.pop();
    for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
      const full = path.join(current, entry.name);
      if (entry.isDirectory()) {
        stack.push(full);
        continue;
      }
      if (entry.name.toLowerCase() === wanted) return full;
    }
  }
  return null;
}

function registerOnPath(binDir) {
  if (process.platform !== "win32" || !isGlobalInstall()) return;
  try {
    const result = ensureInstalledOnPath(binDir);
    const userStatus = result.user && result.user.status;
    if (userStatus === "added" || (result.machine && result.machine.status === "added")) {
      const where =
        result.machine && result.machine.status === "added"
          ? "your user and system Path"
          : "your user Path";
      console.error(`cinedeck: added ${binDir} to ${where}.`);
      console.error(
        "cinedeck: open a new terminal so `cinedeck` is recognized. To use it in this window:"
      );
      console.error(`  cmd:        set "Path=%Path%;${binDir}"`);
      console.error(`  PowerShell: $env:Path += ';${binDir}'`);
    } else if (userStatus === "already") {
      console.error(`cinedeck: ${binDir} is already on your user Path.`);
    }
    if (result.user && result.user.warnings && result.user.warnings.includes("warn:long")) {
      console.error(
        "cinedeck: your Path environment variable is very long; Windows may ignore entries past the limit."
      );
    }
  } catch (err) {
    console.error(`cinedeck: could not update Path (${err.message}).`);
    console.error(
      "cinedeck: add this directory to your Path environment variable, then open a new terminal:"
    );
    console.error(`  ${binDir}`);
  }
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
