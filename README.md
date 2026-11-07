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
| `PaymentsService` | `payments.proto` | Payments | `BUGNET_PAYMENTS_ROUTER_TOKEN` |

`router.proto` imports none of the other three files, so the gateway's generated client holds
none of their services. Errors carry a `google.rpc.ErrorInfo` reason in domain `bugnet.router`.

`BUGNET_PAYMENTS_ROUTER_TOKEN` has the same name and value in Payments' environment and the
router's. It is not `BUGNET_PAYMENTS_TOKEN`, which the gateway sends to Payments.
`PaymentsService` has two calls. `CheckPayer` says whether the router would refuse a payer's order
now. `GetOrder` says what became of an order, so Payments can refund one no event reached it for.

## Generating

`make generate` runs `buf generate`. It writes the Go packages to `gen/go` with `protoc-gen-go`
and `protoc-gen-connect-go` from `PATH`, and JavaScript with TypeScript declarations to `gen/ts`
with `protoc-gen-es` from `node_modules`, so run `pnpm install` first. Imports between the
generated files end in `.js`, which Node's ESM loader needs. Run `make test`, `make lint` and
`make build` for the Go code.

## TypeScript

The ESM-only package `@bugnet/proto` exports three paths, and needs `@bufbuild/protobuf` 2.16 or
a later 2.x beside it.

| Path | Holds | For |
| --- | --- | --- |
| `@bugnet/proto/router/v1` | `router.proto`, with `SandboxRouterService` | The MCP gateway |
| `@bugnet/proto/router/v1/payments` | `payments.proto`, with `PaymentsService` | Payments |
| `@bugnet/proto/vectors/x402/<file>` | One x402 test vector file | The MCP gateway |

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

## Test vectors

`vectors/x402` holds the x402 test cases that the router, `bugnet-egress` and the MCP gateway each
run, so the three never drift. Go code reads them from `vectors.X402`, an `embed.FS` in package
`github.com/oynozan/bugnet-proto/vectors`. TypeScript imports them as JSON.

```ts
import paths from "@bugnet/proto/vectors/x402/paths.json" with { type: "json" };
```

| File | Cases | Run by |
| --- | --- | --- |
| `paths.json` | `paths` of `{path, ok}`, `queries` of `{query, ok}` | Router, egress, gateway |
| `service-ids.json` | `cases` of `{method, host, prefix, id}` | Router, gateway |
| `shared-hosting-suffixes.json` | `suffixes`, a list of names | Router, gateway |

- `paths.json` holds the path and query rules of x402 spec 6.7. `ok` says whether the rule passes
  the case. Egress runs both lists. The router and the gateway run only `paths`, since a service
  line has no query.
- `service-ids.json` holds service ids of x402 spec 5.1. An id is `xs_` and the first 16
  characters of lower-case base32 of SHA-256 of method, host and prefix, joined by a zero byte.
- `shared-hosting-suffixes.json` holds the 16 shared-hosting suffixes of x402 spec 6.16, in its
  order. The router's and the gateway's `shared_hosting_suffixes` defaults equal it.
- Each file is one JSON object with `v` 1. No case appears twice.
- The gateway team owns the content. A change needs both the gateway's and the router's owners.

`go test ./vectors` checks each file: the shape, every id against a reference implementation,
every path and query case against a reference of the 6.7 rule, and the suffix list.
