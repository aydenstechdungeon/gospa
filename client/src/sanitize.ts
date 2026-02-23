import DOMPurify from 'dompurify';

const PURIFY_CONFIG = {
ALLOWED_TAGS: [
'a', 'b', 'br', 'code', 'div', 'em', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
'i', 'img', 'li', 'ol', 'p', 'pre', 'span', 'strong', 'table', 'tbody',
'td', 'th', 'thead', 'tr', 'u', 'ul', 'blockquote', 'hr', 'sub', 'sup',
'small', 'mark', 'del', 'ins', 'abbr', 'cite', 'dfn', 'kbd', 'samp', 'var'
],
ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'class', 'id', 'target', 'rel', 'style'],
ALLOW_DATA_ATTR: true,
FORBID_TAGS: ['script', 'iframe', 'object', 'embed', 'form', 'meta', 'link', 'base', 'applet', 'frame', 'frameset', 'style'],
FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur', 'formaction', 'xlink:href'],
ADD_ATTR: ['target'],
FORCE_BODY: true
};

export function domPurifySanitizer(html: string): string {
    return DOMPurify.sanitize(html, PURIFY_CONFIG) as string;
}
