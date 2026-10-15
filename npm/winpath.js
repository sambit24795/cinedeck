"use strict";

// Windows-only helper: register the installed cinedeck directory in the
// user (and, when appropriate, machine) Path environment variable.
// cmd and PowerShell only resolve `cinedeck` when its directory is on Path.
// npm's global shim directory is often missing from Path, which surfaces as
// "'cinedeck' is not recognized".

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const STATUSES = new Set(["added", "already", "removed", "absent"]);

function normalizeWinDir(dir) {
  return String(dir || "")
    .replace(/\//g, "\\")
    .replace(/\\+$/, "")
    .toLowerCase();
}

function splitPath(value) {
  if (!value) return [];
  return String(value)
    .split(";")
    .filter((entry) => entry !== "");
}

function pathHasDir(pathValue, dir) {
  const target = normalizeWinDir(dir);
  if (!target) return false;
  return splitPath(pathValue).some((entry) => normalizeWinDir(entry) === target);
}

// Append dir if it is not already present. Existing entries are kept verbatim.
function appendDir(pathValue, dir) {
  const clean = String(dir || "").replace(/[\\/]+$/, "");
  if (!clean) return pathValue || "";
  if (pathHasDir(pathValue, clean)) return pathValue || "";
  if (!pathValue) return clean;
  return String(pathValue).replace(/;+$/, "") + ";" + clean;
}

// Drop dir, comparing case-insensitively and ignoring trailing slashes.
function removeDir(pathValue, dir) {
  if (!pathValue) return "";
  const target = normalizeWinDir(dir);
  if (!target) return pathValue;
  const parts = String(pathValue).split(";");
  let removed = false;
  const kept = [];
  for (const entry of parts) {
    if (entry !== "" && normalizeWinDir(entry) === target) {
      removed = true;
      continue;
    }
    kept.push(entry);
  }
  if (!removed) return pathValue;
  return kept.filter((entry) => entry !== "").join(";");
}

function isUnderUserProfile(dir) {
  const profile = process.env.USERPROFILE;
  if (!profile) return false;
  const nDir = normalizeWinDir(dir);
  const nProfile = normalizeWinDir(profile);
  return nDir === nProfile || nDir.startsWith(nProfile + "\\");
}

function isElevated() {
  const result = spawnSync("net", ["session"], {
    stdio: "ignore",
    windowsHide: true,
  });
  return result.status === 0;
}

function isGlobalInstall() {
  const flag = String(process.env.npm_config_global || "").toLowerCase();
  return flag === "true" || flag === "1";
}

// ASCII-only. The directory is passed in CINEDECK_BIN_DIR so non-ASCII
// paths never have to survive PowerShell's command-line encoding.
const PS_SCRIPT = String.raw`
$ErrorActionPreference = 'Stop'
$dir = $env:CINEDECK_BIN_DIR
$action = $env:CINEDECK_PATH_ACTION
$scope = $env:CINEDECK_PATH_SCOPE
if ([string]::IsNullOrWhiteSpace($dir)) { throw 'CINEDECK_BIN_DIR is empty' }
if ([string]::IsNullOrWhiteSpace($action)) { $action = 'add' }
if ([string]::IsNullOrWhiteSpace($scope)) { $scope = 'User' }
$dir = [System.IO.Path]::GetFullPath($dir).TrimEnd('\')

if ($scope -eq 'Machine') {
  $hive = [Microsoft.Win32.Registry]::LocalMachine
  $sub = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment'
} elseif ($scope -eq 'User') {
  $hive = [Microsoft.Win32.Registry]::CurrentUser
  $sub = 'Environment'
} else {
  throw "unknown scope: $scope"
}

$key = $hive.OpenSubKey($sub, $true)
if ($null -eq $key -and $scope -eq 'User') {
  $null = $hive.CreateSubKey($sub)
  $key = $hive.OpenSubKey($sub, $true)
}
if ($null -eq $key) { throw "cannot open $scope Path for write" }

try {
  $hasValue = $false
  $kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
  $current = ''
  foreach ($name in @($key.GetValueNames())) {
    if ($name -ieq 'Path') {
      $hasValue = $true
      $kind = $key.GetValueKind($name)
      $raw = $key.GetValue($name, '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
      if ($null -ne $raw) { $current = [string]$raw }
      break
    }
  }
  if ($hasValue -and $kind -ne [Microsoft.Win32.RegistryValueKind]::String -and $kind -ne [Microsoft.Win32.RegistryValueKind]::ExpandString) {
    throw "Path value has unsupported registry type: $kind"
  }

  $parts = @()
  if ($current -ne '') {
    $parts = @($current -split ';' | Where-Object { $_ -ne '' })
  }
  $norm = $dir.ToLowerInvariant()
  $kept = New-Object System.Collections.Generic.List[string]
  $found = $false
  foreach ($p in $parts) {
    $n = $p.Trim().TrimEnd('\').ToLowerInvariant()
    if ($n -eq $norm) {
      $found = $true
      if ($action -eq 'add') { $null = $kept.Add($p.Trim().TrimEnd('\')) }
    } else {
      $null = $kept.Add($p)
    }
  }

  $changed = $false
  if ($action -eq 'add') {
    if (-not $found) {
      $null = $kept.Add($dir)
      $changed = $true
    }
  } elseif ($action -eq 'remove') {
    if ($found) { $changed = $true }
  } else {
    throw "unknown action: $action"
  }

  if ($changed -and $action -eq 'add') {
    foreach ($p in $parts) {
      $ok = $false
      foreach ($n in $kept) {
        if ($n -eq $p) { $ok = $true; break }
      }
      if (-not $ok) { throw 'refusing to update Path because an existing entry would be lost' }
    }
  }

  if ($changed) {
    $updated = ($kept -join ';')
    $writeKind = if ($hasValue) { $kind } else { [Microsoft.Win32.RegistryValueKind]::ExpandString }
    $key.SetValue('Path', $updated, $writeKind)
    $key.Flush()
    if ($updated.Length -gt 2047) {
      [Console]::Out.WriteLine('warn:long')
    }
    try {
      if (-not ('CinedeckEnvBroadcast' -as [type])) {
        Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public class CinedeckEnvBroadcast {
  [DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
  public static extern IntPtr SendMessageTimeout(IntPtr hWnd, int Msg, IntPtr wParam, string lParam, int fuFlags, int uTimeout, out IntPtr lpdwResult);
}
'@
      }
      $broadcast = [IntPtr]::Zero
      [void][CinedeckEnvBroadcast]::SendMessageTimeout([IntPtr]0xffff, 0x1A, [IntPtr]::Zero, 'Environment', 2, 5000, [ref]$broadcast)
    } catch {
      [Console]::Out.WriteLine('warn:broadcast')
    }
  }

  if ($action -eq 'add') {
    if ($changed) { [Console]::Out.WriteLine('added') } else { [Console]::Out.WriteLine('already') }
  } else {
    if ($changed) { [Console]::Out.WriteLine('removed') } else { [Console]::Out.WriteLine('absent') }
  }
} finally {
  $key.Close()
}
`;

function parseStatus(stdout) {
  const lines = String(stdout || "")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  const status = lines.find((line) => STATUSES.has(line)) || "unknown";
  return {
    status,
    warnings: lines.filter((line) => line.startsWith("warn:")),
  };
}

function runPathScript(dir, scope, action) {
  const scriptPath = path.join(
    os.tmpdir(),
    `cinedeck-path-${process.pid}-${scope}-${action}.ps1`
  );
  fs.writeFileSync(scriptPath, PS_SCRIPT, "utf8");
  try {
    const result = spawnSync(
      "powershell.exe",
      [
        "-NoProfile",
        "-NonInteractive",
        "-ExecutionPolicy",
        "Bypass",
        "-File",
        scriptPath,
      ],
      {
        env: {
          ...process.env,
          CINEDECK_BIN_DIR: dir,
          CINEDECK_PATH_SCOPE: scope,
          CINEDECK_PATH_ACTION: action,
        },
        encoding: "utf8",
        windowsHide: true,
      }
    );
    if (result.error) {
      throw result.error;
    }
    if (result.status !== 0) {
      const detail = (result.stderr || result.stdout || "").trim();
      throw new Error(detail || `powershell exited ${result.status}`);
    }
    const parsed = parseStatus(result.stdout);
    if (!STATUSES.has(parsed.status)) {
      throw new Error(
        `unexpected powershell output: ${(result.stdout || "").trim()}`
      );
    }
    return parsed;
  } finally {
    fs.rmSync(scriptPath, { force: true });
  }
}

function updateScope(dir, scope, action) {
  if (process.platform !== "win32") {
    return { status: "skipped" };
  }
  return runPathScript(dir, scope, action);
}

// Adds binDir to the user Path. A machine-wide Path entry is written only
// when this process is elevated and the binary is not inside the installing
// user's profile (a per-user npm prefix must not be pushed onto every account).
function ensureInstalledOnPath(binDir) {
  if (process.platform !== "win32") {
    return { skipped: true };
  }
  const user = updateScope(binDir, "User", "add");
  let machine = { status: "skipped" };
  if (isElevated() && !isUnderUserProfile(binDir)) {
    try {
      machine = updateScope(binDir, "Machine", "add");
    } catch (err) {
      machine = { status: "error", error: err };
    }
  }
  return { user, machine };
}

function removeInstalledFromPath(binDir) {
  if (process.platform !== "win32") {
    return { skipped: true };
  }
  const user = updateScope(binDir, "User", "remove");
  let machine = { status: "skipped" };
  try {
    machine = updateScope(binDir, "Machine", "remove");
  } catch (err) {
    machine = { status: "error", error: err };
  }
  return { user, machine };
}

module.exports = {
  normalizeWinDir,
  pathHasDir,
  appendDir,
  removeDir,
  isUnderUserProfile,
  isElevated,
  isGlobalInstall,
  parseStatus,
  ensureInstalledOnPath,
  removeInstalledFromPath,
};
