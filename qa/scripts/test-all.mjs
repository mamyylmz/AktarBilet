import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const qaRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoRoot = path.resolve(qaRoot, '..');
const artifacts = path.join(qaRoot, 'artifacts');
const qaProject = process.env.QA_PROJECT_NAME || 'aktarbilet-qa';
fs.mkdirSync(artifacts, { recursive: true });

function run(name, command, args, cwd) {
  const startedAt = new Date().toISOString();
  const result = spawnSync(command, args, { cwd, encoding: 'utf8', env: process.env });
  const output = `${result.stdout || ''}${result.stderr || ''}`;
  fs.writeFileSync(path.join(artifacts, `${name}.log`), output);
  return {
    name,
    startedAt,
    completedAt: new Date().toISOString(),
    exitCode: result.status ?? 1,
    command: [command, ...args].join(' ')
  };
}

const results = [];
results.push(run('environment-up', 'docker', [
  'compose', '--project-name', qaProject, '--file', 'docker-compose.qa.yml',
  'up', '--build', '--detach', '--wait'
], qaRoot));

if (results.at(-1).exitCode === 0) {
  results.push(run('go-tests', 'go', [
    'test', '-count=1', '-coverprofile=qa/artifacts/coverage.out', './internal/...'
  ], repoRoot));
  results.push(run('playwright', 'npx', ['playwright', 'test'], qaRoot));
}

results.push(run('environment-down', 'docker', [
  'compose', '--project-name', qaProject, '--file', 'docker-compose.qa.yml',
  'down', '--remove-orphans'
], qaRoot));

fs.writeFileSync(path.join(artifacts, 'run-summary.json'), JSON.stringify(results, null, 2));
for (const result of results) {
  console.log(`${result.exitCode === 0 ? 'PASS' : 'FAIL'} ${result.name} (${result.command})`);
}

process.exit(results.some((result) => result.exitCode !== 0) ? 1 : 0);
