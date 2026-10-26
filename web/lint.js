import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { transformWithEsbuild } from 'vite';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.join(__dirname, 'src');

const FORBIDDEN_PATTERNS = [
  /qa-validator-hmac-secret/i,
  /9a5553a397e6ed44b0dcba6f20da99aeb60bb50c132c985c23a45a3aa4af6600/i,
];

async function lintDirectory(dir) {
  let errors = 0;
  let filesChecked = 0;

  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      const sub = await lintDirectory(fullPath);
      errors += sub.errors;
      filesChecked += sub.filesChecked;
    } else if (entry.name.endsWith('.js') || entry.name.endsWith('.jsx')) {
      filesChecked++;
      const code = fs.readFileSync(fullPath, 'utf8');

      // 1. Check for embedded secrets
      for (const pattern of FORBIDDEN_PATTERNS) {
        if (pattern.test(code)) {
          console.error(`[LINT ERROR] Forbidden credential pattern detected in ${fullPath}`);
          errors++;
        }
      }

      // 2. Syntax & JSX AST verification via esbuild
      try {
        await transformWithEsbuild(code, fullPath, {
          loader: entry.name.endsWith('.jsx') ? 'jsx' : 'js',
          jsx: 'automatic',
        });
      } catch (err) {
        console.error(`[LINT SYNTAX ERROR] ${fullPath}: ${err.message}`);
        errors++;
      }
    }
  }

  return { errors, filesChecked };
}

console.log('Running AktarBilet Frontend Code Linter (esbuild / AST)...');
const { errors, filesChecked } = await lintDirectory(srcDir);

if (errors > 0) {
  console.error(`\n❌ Linter failed: ${errors} errors found in ${filesChecked} files.`);
  process.exit(1);
}

console.log(`\n✅ Linter passed: ${filesChecked} files analyzed, 0 syntax/security violations.`);

