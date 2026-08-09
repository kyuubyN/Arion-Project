// SPDX-License-Identifier: GPL-3.0-only

import { existsSync, mkdirSync } from 'node:fs';
import { homedir, platform as hostPlatform, arch as hostArch } from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import process from 'node:process';
import { fileURLToPath } from 'node:url';

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const mode = process.argv[2];
const requestedOS = process.argv[3];
const requestedArch = process.argv[4] || 'amd64';

function findGo() {
  const explicit = process.env.ARION_GO_BIN;
  if (explicit && existsSync(explicit)) return explicit;
  const executable = hostPlatform() === 'win32' ? 'go.exe' : 'go';
  const candidates = [
    path.join(homedir(), '.local', 'go', 'bin', executable),
    path.join('/usr/local/go/bin', executable)
  ];
  for (const candidate of candidates) if (existsSync(candidate)) return candidate;
  const probe = spawnSync(executable, ['version'], { stdio: 'ignore' });
  if (probe.status === 0) return executable;
  throw new Error('Go 1.22 ou superior não foi encontrado. Use ARION_GO_BIN para informar o executável.');
}

function normalizedHostOS() {
  if (hostPlatform() === 'win32') return 'windows';
  if (hostPlatform() === 'darwin') return 'darwin';
  return 'linux';
}

function normalizedHostArch() {
  return hostArch() === 'arm64' ? 'arm64' : 'amd64';
}

function build(goBinary, targetOS, targetArch, output, packagePath) {
  mkdirSync(path.dirname(output), { recursive: true });
  const result = spawnSync(goBinary, ['build', '-trimpath', '-o', output, packagePath], {
    cwd: projectRoot,
    env: { ...process.env, GOOS: targetOS, GOARCH: targetArch, CGO_ENABLED: '0' },
    stdio: 'inherit'
  });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}

const goBinary = findGo();
if (mode === 'development' || mode === 'development-backend' || mode === 'development-tools') {
  const targetOS = normalizedHostOS();
  const targetArch = normalizedHostArch();
  const extension = targetOS === 'windows' ? '.exe' : '';
  if (mode !== 'development-tools') build(goBinary, targetOS, targetArch, path.join(projectRoot, 'backend', `arion-backend${extension}`), './backend');
  if (mode !== 'development-backend') build(goBinary, targetOS, targetArch, path.join(projectRoot, 'tools', `arion-provider-validator${extension}`), './cmd/arion-provider-validator');
} else if (mode === 'platform') {
  if (!['linux', 'windows'].includes(requestedOS) || !['amd64', 'arm64'].includes(requestedArch)) {
    throw new Error('Uso: node scripts/go-build.mjs platform <linux|windows> <amd64|arm64>');
  }
  const extension = requestedOS === 'windows' ? '.exe' : '';
  const targetRoot = path.join(projectRoot, 'build', 'native', `${requestedOS}-${requestedArch}`);
  build(goBinary, requestedOS, requestedArch, path.join(targetRoot, 'backend', `arion-backend${extension}`), './backend');
  build(goBinary, requestedOS, requestedArch, path.join(targetRoot, 'tools', `arion-provider-validator${extension}`), './cmd/arion-provider-validator');
} else {
  throw new Error('Modo de build inválido. Use development, development-backend, development-tools ou platform.');
}
