import { client, expect, scenario } from "probitas";
import { endpoints } from "./config.ts";

export default scenario("gRPC echo service", {
  tags: ["grpc", "echo", "integration"],
  stepOptions: { timeout: 15_000 },
})
  .resource("grpc", () =>
    client.grpc.createGrpcClient({
      url: endpoints.grpc,
      metadata: { "x-request-id": "grpc-probitas" },
    }))
  .step("reports serving health status", async ({ resources }) => {
    const res = await resources.grpc.call(
      "grpc.health.v1.Health",
      "Check",
      { service: "" },
    );
    expect(res).ok().dataContains({ status: 1 });
  })
  .step("echoes message and metadata on unary call", async ({ resources }) => {
    const res = await resources.grpc.call(
      "echo.v1.Echo",
      "Echo",
      { message: "ping" },
    );
    expect(res)
      .ok()
      .dataContains({
        message: "ping",
        metadata: { "x-request-id": "grpc-probitas" },
      });
  })
  .step("respects delay and deadline propagation", async ({ resources }) => {
    const res = await resources.grpc.call(
      "echo.v1.Echo",
      "EchoWithDelay",
      { message: "slow", delayMs: 20 },
    );
    expect(res).ok().dataContains({ message: "slow" });

    const deadlineRes = await resources.grpc.call(
      "echo.v1.Echo",
      "EchoDeadline",
      { message: "deadline" },
      { timeout: 1000 },
    );
    const data = deadlineRes.data<{
      hasDeadline: boolean;
      deadlineRemainingMs: number;
    }>();
    if (!data.hasDeadline || data.deadlineRemainingMs <= 0) {
      throw new Error("deadline metadata was not propagated");
    }
  })
  .step("returns request metadata", async ({ resources }) => {
    const res = await resources.grpc.call(
      "echo.v1.Echo",
      "EchoRequestMetadata",
      { keys: ["authorization"] },
      { metadata: { authorization: "Bearer grpc-token" } },
    );
    expect(res).ok().dataContains({
      metadata: { authorization: { values: ["Bearer grpc-token"] } },
    });
  })
  .step("returns trailers and echoes metadata", async ({ resources }) => {
    const res = await resources.grpc.call(
      "echo.v1.Echo",
      "EchoWithTrailers",
      {
        message: "trailers",
        trailers: { "x-trailer": "value" },
      },
      { metadata: { "x-meta": "trailers" } },
    );
    expect(res)
      .ok()
      .dataContains({
        message: "trailers",
        metadata: { "x-meta": "trailers" },
      });
  })
  .step("returns large payloads", async ({ resources }) => {
    const res = await resources.grpc.call(
      "echo.v1.Echo",
      "EchoLargePayload",
      { sizeBytes: 256, pattern: "AB" },
    );
    const data = res.data<{ payload: Uint8Array; actualSize: number }>();
    if (data.actualSize !== 256) {
      throw new Error(`unexpected payload size ${data.actualSize}`);
    }
    if (!(data.payload instanceof Uint8Array) || data.payload.length !== 256) {
      throw new Error("payload bytes missing or incorrect length");
    }
  })
  .step("returns structured errors", async ({ resources }) => {
    try {
      await resources.grpc.call(
        "echo.v1.Echo",
        "EchoError",
        { message: "not found", code: 5, details: "missing" },
      );
      throw new Error("expected gRPC error");
    } catch (err) {
      if (err.code !== 5) {
        throw err;
      }
    }
  })
  .step("returns error details", async ({ resources }) => {
    try {
      await resources.grpc.call(
        "echo.v1.Echo",
        "EchoErrorWithDetails",
        {
          code: 3,
          message: "validation failed",
          details: [
            {
              type: "bad_request",
              fieldViolations: [{ field: "email", description: "invalid" }],
            },
          ],
        },
      );
      throw new Error("expected validation error");
    } catch (err) {
      if (err.code !== 3) {
        throw err;
      }
    }
  })
  .step("streams multiple responses", async ({ resources }) => {
    const messages: string[] = [];
    for await (
      const res of resources.grpc.serverStream(
        "echo.v1.Echo",
        "ServerStream",
        { message: "stream", count: 3, intervalMs: 5 },
      )
    ) {
      expect(res).ok();
      messages.push(res.data<{ message: string }>().message);
    }

    if (messages.length !== 3) {
      throw new Error(`expected 3 messages, received ${messages.length}`);
    }
  })
  .step("aggregates client stream messages", async ({ resources }) => {
    const res = await resources.grpc.clientStream(
      "echo.v1.Echo",
      "ClientStream",
      async function* () {
        yield { message: "one" };
        yield { message: "two" };
        yield { message: "three" };
      }(),
    );
    expect(res).ok().dataContains({ message: "one, two, three" });
  })
  .step("echoes bidirectional stream messages", async ({ resources }) => {
    const received: string[] = [];
    for await (
      const res of resources.grpc.bidiStream(
        "echo.v1.Echo",
        "BidirectionalStream",
        async function* () {
          yield { message: "a" };
          yield { message: "b" };
        }(),
      )
    ) {
      expect(res).ok();
      received.push(res.data<{ message: string }>().message);
      if (received.length === 2) {
        break;
      }
    }
    if (received.join(",") !== "a,b") {
      throw new Error(`unexpected bidi messages ${received}`);
    }
  })
  .build();
