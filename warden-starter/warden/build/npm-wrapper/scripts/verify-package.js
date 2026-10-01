"use strict";
const fs = require("node:fs");
const { parseVersion } = require("../lib/runtime");
const metadata = require("../package.json");
parseVersion(metadata.version);
for (const file of ["bin/warden", "lib/runtime.js", "scripts/verify-package.js", "README.md", "LICENSE"]) {
  const info = fs.lstatSync(file);
  if (!info.isFile() || info.isSymbolicLink()) throw Error(`prepack: invalid required file ${file}`);
}
