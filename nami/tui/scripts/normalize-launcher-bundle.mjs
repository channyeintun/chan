#!/usr/bin/env node

import { chmod, readFile, writeFile } from "node:fs/promises";

const [inputPath, outputPath] = process.argv.slice(2);

if (!inputPath || !outputPath) {
  console.error(
    "usage: normalize-launcher-bundle.mjs <input-bundle> <output-bundle>",
  );
  process.exit(1);
}

const nodeShebang = "#!/usr/bin/env node";
const builtinSpecifiers = ["child_process", "fs", "module", "os", "path", "url"];

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

let source = await readFile(inputPath, "utf8");

if (source.startsWith("#!")) {
  source = source.replace(/^#!.*\r?\n/, `${nodeShebang}\n`);
} else {
  source = `${nodeShebang}\n${source}`;
}

source = source.replace(/^\/\/ @bun\r?\n/, "");

for (const specifier of builtinSpecifiers) {
  const escaped = escapeRegExp(specifier);
  source = source.replace(
    new RegExp(`(from\\s+["'])${escaped}(["'])`, "g"),
    `$1node:${specifier}$2`,
  );
  source = source.replace(
    new RegExp(`(import\\s*\\(\\s*["'])${escaped}(["']\\s*\\))`, "g"),
    `$1node:${specifier}$2`,
  );
  source = source.replace(
    new RegExp(`(require\\(\\s*["'])${escaped}(["']\\s*\\))`, "g"),
    `$1node:${specifier}$2`,
  );
}

await writeFile(outputPath, source);
await chmod(outputPath, 0o755);