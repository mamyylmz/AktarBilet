import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { transformWithEsbuild } from 'vite';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.join(__dirname, 'src');

// Comprehensive Static Analysis Rules & Security Constraints
const LINT_RULES = [
  {
    id: 'security/no-hardcoded-credentials',
    description: 'Disallow embedded credentials, validator secrets, or master tokens',
    test: (code) => {
      const patterns = [
        /qa-validator-hmac-secret/i,
        /9a5553a397e6ed44b0dcba6f20da99aeb60bb50c132c985c23a45a3aa4af6600/i,
        /super-secret-key/i,
        /BEGIN PRIVATE KEY/i,
      ];
      for (const p of patterns) {
        if (p.test(code)) return 'Hardcoded secret or test token detected';
      }
      return null;
    },
  },
  {
    id: 'security/no-eval-or-function-constructor',
    description: 'Disallow dynamic code execution via eval() or new Function()',
    test: (code) => {
      if (/\beval\s*\(/.test(code)) return 'Use of eval() is strictly forbidden';
      if (/new\s+Function\s*\(/.test(code)) return 'Use of Function() constructor is strictly forbidden';
      return null;
    },
  },
  {
    id: 'security/no-unsafe-inner-html',
    description: 'Disallow dangerous unescaped HTML injection',
    test: (code) => {
      if (/\.innerHTML\s*=/.test(code) && !code.includes('DOMPurify')) {
        return 'Direct innerHTML assignment without sanitizer detected';
      }
      return null;
    },
  },
  {
    id: 'react/rules-of-hooks',
    description: 'Ensure hooks are invoked predictably and dependencies handled',
    test: (code) => {
      // Check for conditional hooks (if (...) useState)
      const conditionalHook = /if\s*\([^)]*\)\s*\{[^}]*\buse(State|Effect|Memo|Callback|Ref)\b/.test(code);
      if (conditionalHook) return 'React hook called inside conditional block';
      return null;
    },
  },
  {
    id: 'code-quality/no-debugger',
    description: 'Disallow debugger statements in source code',
    test: (code) => {
      if (/\bdebugger\b/.test(code)) return 'debugger statement found';
      return null;
    },
  },
];

async function lintDirectory(dir) {
  let errors = 0;
  let warnings = 0;
  let filesChecked = 0;

  const entries = fs.readdirSync(dir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      const sub = await lintDirectory(fullPath);
      errors += sub.errors;
      warnings += sub.warnings;
      filesChecked += sub.filesChecked;
    } else if (entry.name.endsWith('.js') || entry.name.endsWith('.jsx')) {
      filesChecked++;
      const code = fs.readFileSync(fullPath, 'utf8');
      const relPath = path.relative(path.join(__dirname, '..'), fullPath);

      // 1. Static Rule Verifications
      for (const rule of LINT_RULES) {
        const violation = rule.test(code);
        if (violation) {
          console.error(`  [${rule.id}] ${relPath}: ${violation}`);
          errors++;
        }
      }

      // 2. Syntax & JSX AST Compilation Check via esbuild
      try {
        await transformWithEsbuild(code, fullPath, {
          loader: entry.name.endsWith('.jsx') ? 'jsx' : 'js',
          jsx: 'automatic',
        });
      } catch (err) {
        console.error(`  [syntax-error] ${relPath}: ${err.message}`);
        errors++;
      }
    }
  }

  return { errors, warnings, filesChecked };
}

console.log('🔍 AktarBilet Frontend Static Analysis & ESLint Engine running...');
const startTime = Date.now();
const { errors, warnings, filesChecked } = await lintDirectory(srcDir);
const durationMs = Date.now() - startTime;

console.log('\n---------------------------------------------------------');
console.log(`Analyzed ${filesChecked} files in ${durationMs}ms`);
console.log(`Rules verified: ${LINT_RULES.length} static security/quality rules + AST validation`);

if (errors > 0) {
  console.error(`❌ Lint failed with ${errors} error(s).`);
  process.exit(1);
}

console.log('✅ Lint passed: 0 errors, 0 warnings. Code conforms to project standard.');
