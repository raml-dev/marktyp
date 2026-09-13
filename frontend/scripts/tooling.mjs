import { createRequire } from 'node:module';
import { dirname, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

// Prefer project-local tooling. The user-provided Yapla checkout is an explicit
// fallback, avoiding new dependencies during this authorized refactor.
const local = createRequire(new URL('../package.json', import.meta.url));
const reference = createRequire(
  resolve(process.env.MARKTYP_TOOLING_ROOT || '../../yapla/frontend', 'package.json'),
);
export function resolveTool(name) {
  try {
    return local.resolve(name);
  } catch {
    return reference.resolve(name);
  }
}
export function loadTool(name) {
  return local(resolveTool(name));
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const operation = process.argv[2];
  const commands = {
    format: [
      'prettier/bin/prettier.cjs',
      [
        '--write',
        'src',
        'tests',
        'scripts',
        'package.json',
        'eslint.config.js',
        '--plugin',
        resolveTool('prettier-plugin-svelte'),
      ],
    ],
    check: ['svelte-check/bin/svelte-check', ['--tsconfig', './tsconfig.json']],
    lint: ['eslint/bin/eslint.js', ['src', 'tests', 'scripts']],
  };
  const command = commands[operation];
  if (!command) throw new Error(`Unknown check: ${operation}`);
  let executable;
  try {
    executable =
      operation === 'lint'
        ? resolve(dirname(resolveTool('eslint/package.json')), 'bin/eslint.js')
        : resolveTool(command[0]);
  } catch {
    if (operation !== 'format')
      throw new Error(
        'Validation tooling unavailable; set MARKTYP_TOOLING_ROOT to the Yapla frontend checkout.',
      );
    executable = resolveTool('prettier/bin/prettier.cjs'.replace('.cjs', '.mjs'));
  }
  const result = spawnSync(process.execPath, [executable, ...command[1]], { stdio: 'inherit' });
  process.exit(result.status ?? 1);
}
