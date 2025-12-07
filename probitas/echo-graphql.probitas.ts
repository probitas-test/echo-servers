import { client, expect, outdent, scenario } from "probitas";
import { endpoints } from "./config.ts";

export default scenario("GraphQL echo API (full coverage)", {
  tags: ["graphql", "echo", "integration"],
  stepOptions: { timeout: 15_000 },
})
  .resource("graphql", () =>
    client.graphql.createGraphqlClient({
      url: endpoints.graphql.http,
      wsEndpoint: endpoints.graphql.ws,
      headers: {
        Authorization: "Bearer probitas-token",
        "X-Custom-Header": "graphql-probitas",
      },
      throwOnError: false,
    }))
  .step("echo query returns message", async ({ resources }) => {
    const res = await resources.graphql.query(
      outdent`
        query ($message: String!) {
          echo(message: $message)
        }
      `,
      { message: "hello from probitas" },
    );
    expect(res).ok().dataContains({ echo: "hello from probitas" });
  })
  .step("echoWithDelay respects delay", async ({ resources }) => {
    const res = await resources.graphql.query(
      outdent`
        query ($message: String!, $delayMs: Int!) {
          echoWithDelay(message: $message, delayMs: $delayMs)
        }
      `,
      { message: "slow", delayMs: 25 },
    );
    expect(res).ok().dataContains({ echoWithDelay: "slow" });
    if (res.duration < 10) {
      throw new Error("echoWithDelay returned too quickly");
    }
  })
  .step("echoError surfaces GraphQL errors", async ({ resources }) => {
    const res = await resources.graphql.query(
      outdent`
        query {
          echoError(message: "test-error")
        }
      `,
    );
    if (res.ok) {
      throw new Error("expected GraphQL error response");
    }
    const error = res.errors?.[0];
    if (!error || error.message !== "test-error") {
      throw new Error(
        `unexpected error payload: ${JSON.stringify(res.errors)}`,
      );
    }
  })
  .step("echoPartialError returns mixed results", async ({ resources }) => {
    const res = await resources.graphql.query(
      outdent`
        query {
          echoPartialError(messages: ["ok", "error here", "fine"]) {
            message
            error
          }
        }
      `,
    );
    expect(res)
      .ok()
      .dataContains({
        echoPartialError: [
          { message: "ok", error: null },
          { message: null, error: "message contains 'error'" },
          { message: "fine", error: null },
        ],
      });
  })
  .step(
    "echoWithExtensions includes timing metadata",
    async ({ resources }) => {
      const res = await resources.graphql.query(
        outdent`
        query {
          echoWithExtensions(message: "extension")
        }
      `,
      );
      expect(res).ok().dataContains({ echoWithExtensions: "extension" });
      if (!res.extensions?.timing || !res.extensions?.tracing) {
        throw new Error("expected extensions timing and tracing");
      }
    },
  )
  .step("echoHeaders exposes request headers", async ({ resources }) => {
    const res = await resources.graphql.query(
      outdent`
        query {
          echoHeaders {
            authorization
            custom(name: "X-Custom-Header")
            all { name value }
          }
        }
      `,
    );
    expect(res)
      .ok()
      .dataContains({
        echoHeaders: {
          authorization: "Bearer probitas-token",
          custom: "graphql-probitas",
        },
      });
  })
  .step("echoNested builds correct depth", async ({ resources }) => {
    const res = await resources.graphql.query(
      outdent`
        query {
          echoNested(message: "nest", depth: 3) {
            value
            child { value child { value } }
          }
        }
      `,
    );
    expect(res)
      .ok()
      .dataContains({
        echoNested: {
          value: "nest (level 1)",
          child: {
            value: "nest (level 2)",
            child: { value: "nest (level 3)" },
          },
        },
      });
  })
  .step("echoList returns requested count", async ({ resources }) => {
    const res = await resources.graphql.query(
      outdent`
        query ($count: Int!, $message: String!) {
          echoList(count: $count, message: $message) {
            index
            message
          }
        }
      `,
      { count: 3, message: "item" },
    );
    expect(res)
      .ok()
      .dataContains({
        echoList: [
          { index: 0, message: "item" },
          { index: 1, message: "item" },
          { index: 2, message: "item" },
        ],
      });
  })
  .step("null and optional values are handled", async ({ resources }) => {
    const nullRes = await resources.graphql.query(
      outdent`
        query {
          echoNull
          optionalValue: echoOptional(message: "maybe", returnNull: false)
          optionalNull: echoOptional(message: "maybe", returnNull: true)
        }
      `,
    );
    expect(nullRes)
      .ok()
      .dataContains({
        echoNull: null,
        optionalValue: "maybe",
        optionalNull: null,
      });
    const data = nullRes.data();
    if (data.optionalValue === null || data.optionalValue === undefined) {
      throw new Error("expected non-null optional value");
    }
  })
  .step(
    "mutations create, update, delete messages",
    async ({ resources, store }) => {
      const create = await resources.graphql.query(
        outdent`
        mutation {
          createMessage(text: "hello") { id text }
        }
      `,
      );
      expect(create).ok().dataContains({ createMessage: { text: "hello" } });
      const messageId = create.data().createMessage.id;
      store.set("messageId", messageId);

      const update = await resources.graphql.query(
        outdent`
        mutation ($id: ID!) {
          updateMessage(id: $id, text: "updated") { id text }
        }
      `,
        { id: messageId },
      );
      expect(update).ok().dataContains({ updateMessage: { text: "updated" } });

      const del = await resources.graphql.query(
        outdent`
        mutation ($id: ID!) {
          deleteMessage(id: $id)
        }
      `,
        { id: messageId },
      );
      expect(del).ok().dataContains({ deleteMessage: true });
    },
  )
  .step("batchCreateMessages returns all messages", async ({ resources }) => {
    const res = await resources.graphql.query(
      outdent`
        mutation {
          batchCreateMessages(texts: ["one", "two"]) { text }
        }
      `,
    );
    expect(res).ok().dataContains({
      batchCreateMessages: [{ text: "one" }, { text: "two" }],
    });
  })
  .step(
    "messageCreated subscription receives new messages",
    async ({ resources }) => {
      const sub = resources.graphql.subscribe(
        outdent`
        subscription {
          messageCreated { text }
        }
      `,
      );
      const iterator = sub[Symbol.asyncIterator]();

      // Start waiting for the message *before* triggering the mutation
      const firstMessagePromise = iterator.next();

      // Allow time for the WebSocket connection to be established
      await new Promise((resolve) => setTimeout(resolve, 500));

      // Now, trigger the event that should produce the message
      await resources.graphql.query(
        outdent`
        mutation {
          createMessage(text: "from-subscription") { id }
        }
      `,
      );

      // Await the promise for the message from the subscription
      const first = await firstMessagePromise;
      if (iterator.return) {
        await iterator.return();
      }
      const payload = first.value?.data()?.messageCreated;
      if (payload?.text !== "from-subscription") {
        throw new Error(
          `unexpected subscription payload: ${JSON.stringify(payload)}`,
        );
      }
    },
  )
  .step("countdown subscription streams numbers", async ({ resources }) => {
    const sub = resources.graphql.subscribe(
      outdent`
        subscription {
          countdown(from: 3)
        }
      `,
    );
    const numbers: number[] = [];

    for await (const res of sub) {
      if (!res.ok) {
        throw new Error("countdown subscription returned error");
      }
      numbers.push(res.data().countdown);
      if (numbers.length === 4) {
        break;
      }
    }

    if (numbers.join(",") !== "3,2,1,0") {
      throw new Error(`unexpected countdown sequence ${numbers}`);
    }
  })
  .step("heartbeat subscription emits timestamps", async ({ resources }) => {
    const sub = resources.graphql.subscribe(
      outdent`
        subscription {
          heartbeat(intervalMs: 200)
        }
      `,
    );
    let count = 0;
    for await (const res of sub) {
      if (!res.ok) {
        throw new Error("heartbeat subscription returned error");
      }
      count++;
      if (count >= 2) {
        break;
      }
    }
    if (count < 2) {
      throw new Error("expected at least two heartbeat events");
    }
  })
  .build();
