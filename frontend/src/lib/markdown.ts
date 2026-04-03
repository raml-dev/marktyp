import MarkdownIt from 'markdown-it';
import markdownItFootnote from 'markdown-it-footnote';
import markdownItTaskLists from 'markdown-it-task-lists';
import TurndownService from 'turndown';
import {gfm} from 'turndown-plugin-gfm';

const markdownRenderer = new MarkdownIt({
  breaks: true,
  html: true,
  linkify: true,
});

markdownRenderer.use(markdownItFootnote);
markdownRenderer.use(markdownItTaskLists as never, {enabled: true, label: true});

const turndown = new TurndownService({
  bulletListMarker: '-',
  codeBlockStyle: 'fenced',
  headingStyle: 'atx',
});

turndown.use(gfm);

turndown.addRule('marktypMermaid', {
  filter: 'marktyp-mermaid',
  replacement: (_content: string, node: Element) => {
    const source = unescapeAttribute(node.getAttribute('source') || '');
    return `\n\n\`\`\`mermaid\n${source}\n\`\`\`\n\n`;
  },
});

turndown.addRule('marktypMathBlock', {
  filter: 'marktyp-math-block',
  replacement: (_content: string, node: Element) => {
    const source = unescapeAttribute(node.getAttribute('source') || '');
    return `\n\n$$\n${source}\n$$\n\n`;
  },
});

turndown.addRule('marktypMathInline', {
  filter: 'marktyp-math-inline',
  replacement: (_content: string, node: Element) => {
    const source = unescapeAttribute(node.getAttribute('source') || '');
    return `$${source}$`;
  },
});

turndown.addRule('fencedCodeBlock', {
  filter: (node: Node) => {
    return node.nodeName === 'PRE' && Boolean((node as Element).querySelector('code'));
  },
  replacement: (_content: string, node: Element) => {
    const code = node.querySelector('code');
    const language = code?.getAttribute('data-language') || '';
    const className = code?.getAttribute('class') || '';
    const classMatch = className.match(/language-([\w-]+)/);
    const resolvedLanguage = language || classMatch?.[1] || '';
    const content = code?.textContent?.replace(/\n$/, '') || '';
    return `\n\n\`\`\`${resolvedLanguage}\n${content}\n\`\`\`\n\n`;
  },
});

turndown.addRule('taskItemCheckbox', {
  filter: (node: Node) => {
    const element = node as HTMLInputElement;
    return node.nodeName === 'INPUT' && element.type === 'checkbox';
  },
  replacement: () => '',
});

export function titleFromMarkdown(markdown: string): string {
  const lines = markdown.split('\n');
  for (const line of lines) {
    const trimmed = line.trim();
    if (trimmed.startsWith('# ')) {
      return trimmed.slice(2).trim();
    }
  }

  return 'Untitled';
}

export function markdownToHtml(markdown: string): string {
  const normalized = normalizeSpecialMarkdown(markdown);
  return markdownRenderer.render(normalized);
}

export function htmlToMarkdown(html: string): string {
  const markdown = turndown.turndown(html);
  return cleanupMarkdown(markdown);
}

function normalizeSpecialMarkdown(markdown: string): string {
  let normalized = markdown;

  normalized = normalized.replace(/```mermaid\s*\n([\s\S]*?)```/g, (_match, source: string) => {
    return `<marktyp-mermaid source="${escapeAttribute(source.trim())}"></marktyp-mermaid>`;
  });

  normalized = normalized.replace(/```([\w-]+)?\s*\n([\s\S]*?)```/g, (_match, language: string, source: string) => {
    const safeLanguage = language?.trim() || '';
    return `<pre><code data-language="${escapeAttribute(safeLanguage)}" class="${safeLanguage ? `language-${escapeAttribute(safeLanguage)}` : ''}">${escapeHtml(source.replace(/\n$/, ''))}</code></pre>`;
  });

  normalized = normalized.replace(/\$\$\s*\n?([\s\S]*?)\n?\$\$/g, (_match, source: string) => {
    return `<marktyp-math-block source="${escapeAttribute(source.trim())}"></marktyp-math-block>`;
  });

  normalized = normalized.replace(/\$([^\n$]+)\$/g, (_match, source: string) => {
    return `<marktyp-math-inline source="${escapeAttribute(source.trim())}"></marktyp-math-inline>`;
  });

  return normalized;
}

function cleanupMarkdown(markdown: string): string {
  return markdown
    .replace(/\n{3,}/g, '\n\n')
    .replace(/^\s+/, '')
    .trim();
}

function escapeAttribute(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/\n/g, '&#10;');
}

function unescapeAttribute(value: string): string {
  return value.replace(/&#10;/g, '\n').replace(/&quot;/g, '"').replace(/&amp;/g, '&');
}

function escapeHtml(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
