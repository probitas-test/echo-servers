# Echo gRPC Protos

These protos are intentionally split into many small files to reproduce a reflection bug in `grpc-reflection-js` 0.3.0. The reflection server only returns the file that contains the requested service (`file_containing_symbol`), and the client is expected to fetch each dependency via `file_by_filename`.

With this layout:

- `echo.proto` imports multiple small proto files (deadline, metadata, payload, response, stream, unary) instead of a single aggregate file.
- `grpc-reflection-js` 0.3.0 incorrectly reuses the same reflection request when fetching multiple filenames, so only a subset of dependencies are returned. This leads to `no such type: .echo.v1.*` when the client fails to resolve missing types.

This setup makes it easy to verify reflection dependency resolution in clients; newer/fixed clients should recursively fetch and include all dependencies before loading descriptors.
