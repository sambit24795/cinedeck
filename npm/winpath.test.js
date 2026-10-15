"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const {
  appendDir,
  pathHasDir,
  removeDir,
  parseStatus,
  isGlobalInstall,
  ensureInstalledOnPath,
} = require("./winpath");

test("pathHasDir is case-insensitive and ignores trailing slashes", () => {
  const value = "C:\\Windows;C:\\Users\\Ada\\AppData\\Roaming\\npm";
  assert.equal(
    pathHasDir(value, "c:\\users\\ada\\appdata\\roaming\\npm\\"),
    true
  );
  assert.equal(pathHasDir(value, "C:\\Users\\Ada\\bin"), false);
  assert.equal(pathHasDir("", "C:\\bin"), false);
});

test("appendDir adds a directory once and preserves existing entries", () => {
  assert.equal(appendDir("", "C:\\Tools\\cinedeck"), "C:\\Tools\\cinedeck");
  assert.equal(
    appendDir("C:\\Windows", "C:\\Tools\\cinedeck"),
    "C:\\Windows;C:\\Tools\\cinedeck"
  );
  assert.equal(
    appendDir("C:\\Windows;", "C:\\Tools\\cinedeck"),
    "C:\\Windows;C:\\Tools\\cinedeck"
  );
  const existing = "C:\\Windows;C:\\Tools\\cinedeck";
  assert.equal(appendDir(existing, "c:\\tools\\cinedeck\\"), existing);
  assert.equal(appendDir("%USERPROFILE%\\bin", "C:\\Tools"), "%USERPROFILE%\\bin;C:\\Tools");
});

test("removeDir drops only the cinedeck directory", () => {
  assert.equal(
    removeDir("C:\\Windows;C:\\Tools\\cinedeck;C:\\Other", "c:\\tools\\cinedeck\\"),
    "C:\\Windows;C:\\Other"
  );
  assert.equal(removeDir("C:\\Windows", "C:\\Tools\\cinedeck"), "C:\\Windows");
  assert.equal(removeDir("", "C:\\Tools\\cinedeck"), "");
  assert.equal(
    removeDir("C:\\Tools\\cinedeck-extra", "C:\\Tools\\cinedeck"),
    "C:\\Tools\\cinedeck-extra"
  );
});

test("parseStatus reads installer output", () => {
  assert.deepEqual(parseStatus("added\n"), { status: "added", warnings: [] });
  assert.deepEqual(parseStatus("warn:long\r\nadded\r\n"), {
    status: "added",
    warnings: ["warn:long"],
  });
  assert.equal(parseStatus("nope").status, "unknown");
});

test("ensureInstalledOnPath does not touch the environment off Windows", () => {
  if (process.platform === "win32") return;
  assert.deepEqual(ensureInstalledOnPath("C:\\Tools\\cinedeck"), {
    skipped: true,
  });
});

test("isGlobalInstall follows npm's npm_config_global flag", () => {
  const previous = process.env.npm_config_global;
  try {
    process.env.npm_config_global = "true";
    assert.equal(isGlobalInstall(), true);
    process.env.npm_config_global = "1";
    assert.equal(isGlobalInstall(), true);
    process.env.npm_config_global = "";
    assert.equal(isGlobalInstall(), false);
    delete process.env.npm_config_global;
    assert.equal(isGlobalInstall(), false);
  } finally {
    if (previous === undefined) delete process.env.npm_config_global;
    else process.env.npm_config_global = previous;
  }
});
