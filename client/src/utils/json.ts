/**
 * Safe JSON parsing utility to prevent Prototype Pollution.
 * Filters out dangerous keys like __proto__, constructor, and prototype.
 */
export function safeJSONParse(text: string): any {
  return JSON.parse(text, (key, value) => {
    if (key === "__proto__" || key === "constructor" || key === "prototype") {
      return undefined;
    }
    return value;
  });
}
