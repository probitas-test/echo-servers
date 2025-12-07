export const endpoints = {
  http: Deno.env.get("ECHO_HTTP_BASE_URL") ?? "http://localhost:18080",
  grpc: Deno.env.get("ECHO_GRPC_ADDRESS") ?? "localhost:50051",
  graphql: {
    http: Deno.env.get("ECHO_GRAPHQL_HTTP_URL") ??
      "http://localhost:14000/graphql",
    ws: Deno.env.get("ECHO_GRAPHQL_WS_URL") ??
      "ws://localhost:14000/graphql",
  },
  connectrpc: Deno.env.get("ECHO_CONNECTRPC_BASE_URL") ??
    "http://localhost:18081",
};
