import { client, expect, scenario } from "probitas";
import { endpoints } from "./config.ts";

export default scenario("Connect RPC echo service (full coverage)", {
  tags: ["connectrpc", "echo", "integration"],
  stepOptions: { timeout: 15_000 },
})
  .resource("connect", () =>
    client.connectrpc.createConnectRpcClient({
      url: endpoints.connectrpc,
    }))
  .step("reports serving health status", async ({ resources }) => {
    const res = await resources.connect.call(
      "grpc.health.v1.Health",
      "Check",
      { service: "" },
    );
    expect(res).ok().dataContains({ status: 1 });
  })
  .step("echo returns message", async ({ resources }) => {
    const res = await resources.connect.call(
      "echo.v1.Echo",
      "Echo",
      { message: "connect echo" },
    );
    expect(res)
      .ok()
      .dataContains({ message: "connect echo" });
  })
  .step("supports delay and deadline reporting", async ({ resources }) => {
    const delayed = await resources.connect.call(
      "echo.v1.Echo",
      "EchoWithDelay",
      { message: "delayed", delayMs: 15 },
    );
    expect(delayed).ok().dataContains({ message: "delayed" });

    const deadline = await resources.connect.call(
      "echo.v1.Echo",
      "EchoDeadline",
      { message: "deadline" },
      { timeout: 1000 },
    );
    const data = deadline.data<{
      hasDeadline: boolean;
      deadlineRemainingMs: number;
    }>();
    if (!data.hasDeadline || data.deadlineRemainingMs <= 0) {
      throw new Error("deadline metadata missing");
    }
  })
  .step("returns request metadata and trailers", async ({ resources }) => {
    const res = await resources.connect.call(
      "echo.v1.Echo",
      "EchoWithTrailers",
      {
        message: "trailers",
        trailers: { "x-trailer": "value" },
      },
      { metadata: { "x-meta": "connect" } },
    );
    expect(res)
      .ok()
      .dataContains({
        message: "trailers",
        metadata: { "X-Meta": "connect" },
      });
  })
  .step("returns request metadata only when requested keys match", async ({ resources }) => {
    const res = await resources.connect.call(
      "echo.v1.Echo",
      "EchoRequestMetadata",
      { keys: ["authorization"] },
      { metadata: { authorization: "Bearer connect-token" } },
    );
    expect(res).ok().dataContains({
      metadata: { authorization: { values: ["Bearer connect-token"] } },
    });
  })
  .step("handles large payloads", async ({ resources }) => {
    const res = await resources.connect.call(
      "echo.v1.Echo",
      "EchoLargePayload",
      { sizeBytes: 128, pattern: "XY" },
    );
    const data = res.data<{ payload: Uint8Array; actualSize: number }>();
    if (data.actualSize !== 128 || data.payload.length !== 128) {
      throw new Error(`unexpected payload size ${data.actualSize}/${data.payload.length}`);
    }
  })
  .step("returns structured errors", async ({ resources }) => {
    try {
      await resources.connect.call(
        "echo.v1.Echo",
        "EchoError",
        { message: "permission", code: 7, details: "denied" },
      );
      throw new Error("expected permission error");
    } catch (err) {
      if (err.code !== 7) {
        throw err;
      }
    }
  })
  .step("returns error details", async ({ resources }) => {
    try {
      await resources.connect.call(
        "echo.v1.Echo",
        "EchoErrorWithDetails",
        {
          code: 14,
          message: "retry later",
          details: [{ type: "retry_info", retryDelayMs: 10 }],
        },
      );
      throw new Error("expected retry error");
    } catch (err) {
      if (err.code !== 14) {
        throw err;
      }
    }
  })
  .step("server streaming returns requested count", async ({ resources }) => {
    const messages: string[] = [];
    for await (
      const res of resources.connect.serverStream(
        "echo.v1.Echo",
        "ServerStream",
        { message: "stream", count: 2, intervalMs: 10 },
      )
    ) {
      expect(res).ok();
      messages.push(res.data<{ message: string }>().message);
    }

    if (messages.length !== 2) {
      throw new Error(`expected 2 messages, received ${messages.length}`);
    }
  })
  .step("aggregates client stream messages", async ({ resources }) => {
    const res = await resources.connect.clientStream(
      "echo.v1.Echo",
      "ClientStream",
      async function* () {
        yield { message: "alpha" };
        yield { message: "beta" };
      }(),
    );
    expect(res).ok().dataContains({ message: "alpha, beta" });
  })
  .step("echoes bidirectional stream messages", async ({ resources }) => {
    const received: string[] = [];
    for await (
      const res of resources.connect.bidiStream(
        "echo.v1.Echo",
        "BidirectionalStream",
        async function* () {
          yield { message: "x" };
          yield { message: "y" };
        }(),
      )
    ) {
      expect(res).ok();
      received.push(res.data<{ message: string }>().message);
      if (received.length === 2) {
        break;
      }
    }
    if (received.join(",") !== "x,y") {
      throw new Error(`unexpected bidi messages ${received}`);
    }
  })
  .build();
