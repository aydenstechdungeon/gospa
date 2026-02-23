export function simpleSanitizer(html: string): string {
    const div = document.createElement('div');
    div.innerHTML = html;
    const scripts = div.getElementsByTagName('script');
    let i = scripts.length;
    while (i--) {
      scripts[i].parentNode?.removeChild(scripts[i]);
    }
    return div.innerHTML;
}
