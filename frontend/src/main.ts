import 'flowbite/dist/flowbite.min.css';
import 'katex/dist/katex.min.css';
import 'highlight.js/styles/github-dark.css';
import './style.css';
import App from './App.svelte';

const app = new App({
  target: document.getElementById('root')!,
});

export default app;
