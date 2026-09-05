#!/usr/bin/env node
"use strict";

const https = require("https");
const fs = require("fs");
const path = require("path");
const os = require("os");
const { execSync } = require("child_process");

const VERSION = process.env.npm_package_version || "0.1.0";

const platform = os.platform(); // 'linux', 'darwin', 'windows'
const arch = os.arch();        // 'x64', 'arm64', etc.

const osName = platform === "win32" ? "windows" : platform;
const archName = arch === "x64" ? "amd64" : arch;

const fileName = `warden-${osName}-${archName}${platform === "win32" ? ".exe" : ""}`;
const url = `https://github.com/warden-sandbox/warden/releases/download/v${VERSION}/${fileName}`;

const binDir = path.join(__dirname, "bin");
const binPath = path.join(binDir, platform === "win32" ? "warden.exe" : "warden");

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    https.get(url, { timeout: 30000 }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        download(res.headers.location, dest).then(resolve).catch(reject);
        return;
      }
      if (res.statusCode !== 200) {
        reject(new Error(`HTTP ${res.statusCode} downloading ${url}`));
        return;
      }
      res.pipe(file);
      file.on("finish", () => {
        file.close();
        resolve();
      });
    }).on("error", (err) => {
      fs.unlink(dest, () => {});
      reject(err);
    });
  });
}

(async () => {
  if (!fs.existsSync(binDir)) fs.mkdirSync(binDir, { recursive: true });

  if (!fs.existsSync(binPath)) {
    try {
      console.log(`Downloading ${fileName} from ${url}`);
      await download(url, binPath);
      console.log(`Installed to ${binPath}`);
    } catch (err) {
      console.error(`Failed to download binary: ${err.message}`);
      process.exit(1);
    }
  }

  if (platform !== "win32") {
    fs.chmodSync(binPath, 0o755);
  }
})();
