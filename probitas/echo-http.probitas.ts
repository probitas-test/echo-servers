import { client, expect, scenario } from "probitas";
import { endpoints } from "./config.ts";

export default scenario("HTTP echo endpoints (full coverage)", {
  tags: ["http", "echo", "integration"],
  stepOptions: { timeout: 10_000 },
})
  .resource("http", () =>
    client.http.createHttpClient({
      url: endpoints.http,
    }))
  .step("responds to health check", async ({ resources }) => {
    const res = await resources.http.get("/health");
    expect(res).ok().status(200).dataContains({ status: "ok" });
  })
  .step("echoes query parameters", async ({ resources }) => {
    const res = await resources.http.get("/get", {
      query: { hello: "world", answer: "42" },
    });
    expect(res)
      .ok()
      .status(200)
      .dataContains({
        args: { hello: "world", answer: "42" },
      });
  })
  .step("echoes JSON payloads", async ({ resources }) => {
    const payload = { message: "probitas-http", count: 2 };
    const res = await resources.http.post("/post", payload);
    expect(res)
      .ok()
      .status(200)
      .dataContains({
        json: payload,
      });
  })
  .step("echoes form payloads", async ({ resources }) => {
    const res = await resources.http.post(
      "/post",
      "name=http&lang=go",
      { headers: { "Content-Type": "application/x-www-form-urlencoded" } },
    );
    expect(res).ok().status(200).dataContains({
      form: { name: "http", lang: "go" },
    });
  })
  .step("supports PUT and PATCH", async ({ resources }) => {
    const putRes = await resources.http.put("/put", { updated: true });
    expect(putRes).ok().status(200).dataContains({
      json: { updated: true },
    });

    const patchRes = await resources.http.patch("/patch", { patched: 1 });
    expect(patchRes).ok().status(200).dataContains({
      json: { patched: 1 },
    });
  })
  .step("supports DELETE with query", async ({ resources }) => {
    const res = await resources.http.delete("/delete", {
      query: { id: "123" },
    });
    expect(res)
      .ok()
      .status(200)
      .dataContains({ url: "/delete?id=123", args: { id: "123" } });
  })
  .step("validates bearer authentication", async ({ resources }) => {
    const res = await resources.http.get("/bearer", {
      headers: { Authorization: "Bearer probitas-token" },
    });
    expect(res)
      .ok()
      .status(200)
      .dataContains({
        authenticated: true,
        token: "probitas-token",
      });
  })
  .step("handles basic and hidden basic auth", async ({ resources }) => {
    const basicToken = btoa("user:pass");
    const basic = await resources.http.get("/basic-auth/user/pass", {
      headers: { Authorization: `Basic ${basicToken}` },
    });
    expect(basic).ok().status(200).dataContains({
      authenticated: true,
      user: "user",
    });

    const hidden = await resources.http.get(
      "/hidden-basic-auth/user/pass",
      { headers: { Authorization: `Basic ${basicToken}` } },
    );
    expect(hidden).ok().status(200).dataContains({
      authenticated: true,
      user: "user",
    });
  })
  .step("echoes custom headers", async ({ resources }) => {
    const res = await resources.http.get("/headers", {
      headers: {
        "X-Request-Id": "http-req-1",
        "X-Custom": "value",
      },
    });
    expect(res)
      .ok()
      .status(200)
      .dataContains({
        headers: {
          "X-Request-Id": "http-req-1",
          "X-Custom": "value",
        },
      });
  })
  .step("returns requested status codes", async ({ resources }) => {
    const res = await resources.http.get("/status/418", {
      throwOnError: false,
    });
    expect(res).status(418);
  })
  .step("honors delay endpoint", async ({ resources }) => {
    const res = await resources.http.get("/delay/1");
    expect(res).ok().status(200);
    if (res.duration < 500 || res.duration > 4000) {
      throw new Error(`unexpected delay duration ${res.duration}ms`);
    }
  })
  .step("manages cookies", async ({ resources }) => {
    const res = await resources.http.get("/cookies", {
      headers: { Cookie: "session=abc123; theme=light" },
    });
    expect(res)
      .ok()
      .status(200)
      .dataContains({
        cookies: { session: "abc123", theme: "light" },
      });
  })
  .step("follows redirects", async ({ resources }) => {
    const res = await resources.http.get("/redirect/2");
    expect(res).ok().status(200).dataContains({ redirected: true });
  })
  .step("serves raw bytes", async ({ resources }) => {
    const res = await resources.http.get("/bytes/8");
    expect(res).ok().contentType(/application\/octet-stream/);
    expect(res.body?.byteLength ?? 0).toBe(8);
  })
  .step("streams newline-delimited JSON", async ({ resources }) => {
    const res = await resources.http.get("/stream/3");
    expect(res).ok().status(200);
    const lines = res.text()?.trim().split("\n") ?? [];
    if (lines.length !== 3) {
      throw new Error(`expected 3 lines, got ${lines.length}`);
    }
  })
  .step("drips bytes over time", async ({ resources }) => {
    const res = await resources.http.get("/drip", {
      query: { duration: "1", numbytes: "5", delay: "0" },
    });
    expect(res).ok().status(200);
    const body = res.text() ?? "";
    if (body.length !== 5) {
      throw new Error(`expected 5 bytes, got ${body.length}`);
    }
  })
  .step("reports client ip and user agent", async ({ resources }) => {
    const ipRes = await resources.http.get("/ip");
    expect(ipRes).ok();
    const origin = ipRes.data<{ origin?: string }>()?.origin;
    if (!origin) {
      throw new Error("origin not returned");
    }

    const uaRes = await resources.http.get("/user-agent", {
      headers: { "User-Agent": "probitas-http-client" },
    });
    expect(uaRes).ok().dataContains({ "user-agent": "probitas-http-client" });
  })
  .build();
