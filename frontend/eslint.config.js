import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import ts from 'typescript-eslint';

export default [
  { ignores: ['dist/**', 'node_modules/**', 'wailsjs/**'] },
  js.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs['flat/recommended'],
  { languageOptions: { globals: { ...globals.browser, ...globals.node } }, rules: { 'no-undef': 'off' } },
  { files: ['**/*.svelte'], languageOptions: { parserOptions: { parser: ts.parser } } },
];
