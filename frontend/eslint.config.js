import { loadTool } from './scripts/tooling.mjs';
const js = loadTool('@eslint/js');
const ts = loadTool('typescript-eslint');
const svelte = loadTool('eslint-plugin-svelte');
const globals = loadTool('globals');

export default [
  { ignores: ['dist/**', 'node_modules/**', 'wailsjs/**'] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs['flat/recommended'],
  { languageOptions: { globals: { ...globals.browser, ...globals.node } }, rules: { 'no-undef': 'off' } },
  { files: ['**/*.svelte'], languageOptions: { parserOptions: { parser: ts.parser } } },
];
