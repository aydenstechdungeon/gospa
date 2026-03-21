import { describe, expect, it } from "bun:test";
import { SSEClient } from "./sse.ts";

describe("SSEClient", () => {
  it("rejects authentication headers that would leak into the URL", () => {
    const client = new SSEClient({
      url: "/events",
      headers: { Authorization: "Bearer demo-token" },
    });

    expect(() => client.connect()).toThrow(
      "SSE authentication headers are not supported because EventSource would expose them in the URL. Use same-origin cookies or a short-lived ticket instead.",
    );
  });
});
