export interface TrustedHTMLValue {
  readonly __gospaTrustedHTML: true;
  readonly html: string;
}

function isTrustedHTMLValue(value: unknown): value is TrustedHTMLValue {
  return Boolean(
    value &&
    typeof value === "object" &&
    (value as { __gospaTrustedHTML?: unknown }).__gospaTrustedHTML === true &&
    typeof (value as { html?: unknown }).html === "string",
  );
}

export function trustedHTML(html: string): TrustedHTMLValue {
  return {
    __gospaTrustedHTML: true,
    html,
  };
}

function escapeHTML(input: string): string {
  return input
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

export function toHTMLString(value: unknown): string {
  if (isTrustedHTMLValue(value)) {
    return value.html;
  }

  return escapeHTML(String(value ?? ""));
}
