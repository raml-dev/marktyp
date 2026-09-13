const allowedTags = new Set(
  'a abbr b blockquote br code dd del details div dl dt em h1 h2 h3 h4 h5 h6 hr i img input kbd label li mark ol p pre s samp section small span strong sub summary sup table tbody td th thead tr ul marktyp-math-inline marktyp-math-block marktyp-mermaid'.split(
    ' ',
  ),
);
const allowedAttributes = new Set(
  'alt checked class colspan data-language data-marktyp-inline-boundary data-marktyp-source height href id label role rowspan source src start title type width'.split(
    ' ',
  ),
);

export function isSafeUrl(value: string): boolean {
  if (/^[a-z]:[\\/]/i.test(value)) return true;
  // URL scheme validation must include embedded ASCII controls used in obfuscated protocols.
  // eslint-disable-next-line no-control-regex
  const normalized = value.replace(/[\u0000-\u0020\u007f]/g, '');
  if (!/^[a-z][a-z\d+.-]*:/i.test(normalized)) return true;
  return (
    /^(https?:|mailto:|file:)/i.test(normalized) ||
    /^data:image\/(png|jpe?g|gif|webp);base64,/i.test(normalized)
  );
}

// Markdown is untrusted input inside a desktop webview with privileged RPC access.
export function sanitizeHtml(html: string): string {
  const template = document.createElement('template');
  template.innerHTML = html;
  for (const element of Array.from(template.content.querySelectorAll('*'))) {
    if (!allowedTags.has(element.localName)) {
      element.remove();
      continue;
    }
    for (const attribute of Array.from(element.attributes)) {
      const name = attribute.name;
      if (
        !allowedAttributes.has(name) ||
        ((name === 'href' || name === 'src') && !isSafeUrl(attribute.value))
      ) {
        element.removeAttribute(name);
      }
    }
    if (element.localName === 'input') {
      element.removeAttribute('disabled');
      if (element.getAttribute('type') !== 'checkbox') {
        element.remove();
        continue;
      }
    }
  }
  return template.innerHTML;
}
