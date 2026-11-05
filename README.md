# bugnet-proto

Protobuf and Connect API definitions shared by the Bugnet services.

`make generate` runs `buf generate`. It writes the Go packages to `gen/go` with `protoc-gen-go`
and `protoc-gen-connect-go` from `PATH`, and JavaScript with TypeScript declarations to `gen/ts`
with `protoc-gen-es` from `node_modules`, so run `pnpm install` first. Run `make test`,
`make lint` and `make build` for the Go code.

## TypeScript

The ESM-only package `@bugnet/proto` exports `@bugnet/proto/router/v1`, the router's messages and
`SandboxRouterService`, and needs `@bufbuild/protobuf` 2.16 or a later 2.x beside it. Use it from
a sibling checkout:

```json
"dependencies": {
  "@bufbuild/protobuf": "2.16.0",
  "@bugnet/proto": "link:../bugnet-proto"
}
```

```ts
import { createClient } from "@connectrpc/connect";
import { SandboxRouterService } from "@bugnet/proto/router/v1";

const router = createClient(SandboxRouterService, transport);
```

Run `pnpm install` here once too. Node follows the link to this checkout, so the generated code
loads its own `@bufbuild/protobuf` from this `node_modules`, unless Node runs with
`--preserve-symlinks`.
