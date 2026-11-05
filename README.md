# bugnet-proto

Protobuf and Connect API definitions shared by the Bugnet services.

## Services

Every file is in package `bugnet.router.v1` under `proto/bugnet/router/v1`. Each service has one
caller, which sends that service's own bearer token. The router refuses any other service's token,
and admits no calls to a service whose token is empty.

| Service | File | Caller | Token variable |
| --- | --- | --- | --- |
| `SandboxRouterService` | `router.proto` | MCP gateway, `bugnet-dev` | `BUGNET_INTERNAL_TOKEN` |
| `EdgeService` | `edge.proto` | `bugnet-edge` | `BUGNET_EDGE_TOKEN` |
| `SecretsRouteService` | `edge.proto` | `bugnet-secrets` | `BUGNET_SECRETS_TOKEN` |
| `SshGatewayService` | `edge.proto` | `bugnet-ssh` | `BUGNET_SSH_TOKEN` |
| `EgressService` | `edge.proto` | `bugnet-egress` | `BUGNET_EGRESS_TOKEN` |
| `OperatorService` | `operator.proto` | `bugnet-dev`, operators only | `BUGNET_OPERATOR_TOKEN` |
| `PaymentsService` | `payments.proto` | Payments | `BUGNET_PAYMENTS_TOKEN` |

`router.proto` imports none of the other three files, so the gateway's generated client holds
none of their services. Errors carry a `google.rpc.ErrorInfo` reason in domain `bugnet.router`.

## Generating

`make generate` runs `buf generate`. It writes the Go packages to `gen/go` with `protoc-gen-go`
and `protoc-gen-connect-go` from `PATH`, and JavaScript with TypeScript declarations to `gen/ts`
with `protoc-gen-es` from `node_modules`, so run `pnpm install` first. Run `make test`,
`make lint` and `make build` for the Go code.

## TypeScript

The ESM-only package `@bugnet/proto` exports two paths, and needs `@bufbuild/protobuf` 2.16 or a
later 2.x beside it.

| Path | Holds | For |
| --- | --- | --- |
| `@bugnet/proto/router/v1` | `router.proto`, with `SandboxRouterService` | The MCP gateway |
| `@bugnet/proto/router/v1/payments` | `payments.proto`, with `PaymentsService` | Payments |

`gen/ts` also holds `edge.proto` and `operator.proto`, since `buf generate` runs every plugin on
every file. The package exports neither, because only Go programs call those services. Use the
package from a sibling checkout:

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

```ts
import { PaymentsService } from "@bugnet/proto/router/v1/payments";

const payments = createClient(PaymentsService, transport);
```

Run `pnpm install` here once too. Node follows the link to this checkout, so the generated code
loads its own `@bufbuild/protobuf` from this `node_modules`, unless Node runs with
`--preserve-symlinks`.
