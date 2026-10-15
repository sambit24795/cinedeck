"use strict";

// Removes the directory install.js added to the Windows Path environment
// variable. npm runs this before the package files are deleted.

const path = require("node:path");
const { removeInstalledFromPath } = require("./winpath");

function main() {
  if (process.platform !== "win32") return;
  const binDir = path.join(__dirname, "bin");
  try {
    const result = removeInstalledFromPath(binDir);
    if (result.user && result.user.status === "removed") {
      console.error(
        `cinedeck: removed ${binDir} from your user Path. Open a new terminal to pick up the change.`
      );
    }
  } catch (err) {
    console.error(`cinedeck: could not update Path (${err.message}).`);
  }
}

main();
