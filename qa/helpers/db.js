import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const qaRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const composeFile = path.join(qaRoot, 'docker-compose.qa.yml');
const qaProject = process.env.QA_PROJECT_NAME || 'aktarbilet-qa';

export function sql(statement) {
  return execFileSync('docker', [
    'compose',
    '--project-name', qaProject,
    '--file', composeFile,
    'exec', '-T', 'postgres',
    'psql', '-X', '-q', '-v', 'ON_ERROR_STOP=1', '-A', '-t',
    '-U', 'aktarbilet_qa', '-d', 'aktarbilet_qa',
    '-c', statement
  ], { encoding: 'utf8' }).trim();
}

export function scalar(statement) {
  const value = sql(statement).split('\n').filter(Boolean).at(-1) ?? '';
  return value.trim();
}

export function rows(statement) {
  const output = sql(statement);
  return output ? output.split('\n').filter(Boolean) : [];
}
