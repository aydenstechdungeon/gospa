import DOMPurify from 'dompurify';

const PURIFY_CONFIG = {
    ALLOWED_TAGS: [
        // Semantic HTML5
        'header', 'footer', 'nav', 'main', 'section', 'article', 'aside', 'figure', 'figcaption',
        // Interactive elements
        'button', 'details', 'summary', 'dialog',
        // Standard elements
        'a', 'b', 'br', 'code', 'div', 'em', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
        'i', 'img', 'li', 'ol', 'p', 'pre', 'span', 'strong', 'table', 'tbody',
        'td', 'th', 'thead', 'tr', 'u', 'ul', 'blockquote', 'hr', 'sub', 'sup',
        'small', 'mark', 'del', 'ins', 'abbr', 'cite', 'dfn', 'kbd', 'samp', 'var',
        'svg', 'path', 'rect', 'circle', 'line', 'polyline', 'polygon', 'ellipse', 'style',
        // Additional useful elements
        'input', 'textarea', 'select', 'option', 'optgroup', 'label', 'form', 'fieldset', 'legend',
        'dl', 'dt', 'dd', 'address', 'time', 'picture', 'source', 'audio', 'video', 'track',
        'canvas', 'data', 'meter', 'progress', 'output'
    ],
    ALLOWED_ATTR: [
        'href', 'src', 'alt', 'title', 'class', 'id', 'target', 'rel', 'style',
        'xmlns', 'viewBox', 'fill', 'stroke', 'stroke-width', 'stroke-linecap', 'stroke-linejoin',
        'd', 'x', 'y', 'width', 'height', 'rx', 'ry', 'points', 'cx', 'cy', 'r'
    ],
    ALLOW_DATA_ATTR: true,
    FORBID_TAGS: ['script', 'iframe', 'object', 'embed', 'form', 'meta', 'link', 'base', 'applet', 'frame', 'frameset'],
    FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur', 'formaction', 'xlink:href'],
    ADD_ATTR: ['target'],
    FORCE_BODY: true
};

export function domPurifySanitizer(html: string): string {
    return DOMPurify.sanitize(html, PURIFY_CONFIG) as string;
}
