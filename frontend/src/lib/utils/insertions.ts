export function markdownDestination(value: string): string {
  const escaped = value.replace(/\\/g, '\\\\').replace(/\(/g, '\\(').replace(/\)/g, '\\)');
  return /\s/.test(escaped) ? `<${escaped.replace(/</g, '%3C').replace(/>/g, '%3E')}>` : escaped;
}

export function createMarkdownTable(rows: number, columns: number): string {
  const headers = Array.from({ length: columns }, (_, index) => `Column ${index + 1}`);
  const heading = `| ${headers.join(' | ')} |`;
  const separator = `| ${headers.map(() => '---').join(' | ')} |`;
  const body = Array.from({ length: rows - 1 }, () => `| ${headers.map(() => 'Cell').join(' | ')} |`).join(
    '\n',
  );
  return `\n${heading}\n${separator}\n${body}\n`;
}

export function createHTMLTable(rows: number, columns: number): string {
  const headers = Array.from({ length: columns }, (_, index) => `Column ${index + 1}`);
  const heading = headers
    .map((value, index) => `<th${index === 0 ? ' data-marktyp-edit-target' : ''}>${value}</th>`)
    .join('');
  const body = Array.from(
    { length: rows - 1 },
    () => `<tr>${headers.map(() => '<td>Cell</td>').join('')}</tr>`,
  ).join('');
  return `<table><thead><tr>${heading}</tr></thead><tbody>${body}</tbody></table><p><br></p>`;
}
