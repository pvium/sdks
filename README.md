# Pvium SDKs

Pvium provides programmable zero-custody stablecoin payments for the global gig
economy.

This monorepo contains the official Pvium SDKs for TypeScript, Python, and Go.
Each SDK exposes Pvium API clients plus helpers for OAuth invite links, webhooks,
batch payouts, scheduled payouts, escrow payouts, Merkle proofs, and signing
flows used by Pvium payment workflows.

- Developer documentation: https://pvium.gitbook.io/documentation
- Pvium website: https://pvium.com
- Pvium sandbox: https://sandbox.pvium.com

## SDKs

| SDK | Package | Install | Docs |
| --- | --- | --- | --- |
| TypeScript / Node.js | `@pvium/sdk` | `npm install @pvium/sdk` | [node-sdk/README.md](node-sdk/README.md) |
| Python | `pvium` | `pip install pvium` | [python-sdk/README.md](python-sdk/README.md) |
| Go | `github.com/pvium/sdks/go-sdk` | `go get github.com/pvium/sdks/go-sdk` | [go-sdk/README.md](go-sdk/README.md) |

## Quick Starts

TypeScript:

```ts
import { PviumSdk } from "@pvium/sdk";

const pvium = PviumSdk.init({
  environment: "sandbox",
  apiKey: process.env.PVIUM_API_KEY as string,
});

const invoices = await pvium.endpoints.listInvoices();
console.log(invoices.meta.success, invoices.data.length);
```

Python:

```python
from pvium_sdk import PviumSdk, PviumSdkConfig

pvium = PviumSdk.init(
    PviumSdkConfig(
        environment="sandbox",
        apiKey="your_api_key",
        clientId="your_client_id",
    )
)

invoices = pvium.endpoints.listInvoices()
print(invoices)
```

Go:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	pvium "github.com/pvium/sdks/go-sdk"
)

func main() {
	sdk := pvium.Init(pvium.Config{
		Environment: pvium.EnvironmentSandbox,
		APIKey:      os.Getenv("PVIUM_API_KEY"),
	})

	invoices, err := sdk.Endpoints.ListInvoices(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(invoices.Meta.Success, len(invoices.Data))
}
```

## Repository Layout

```text
.
  node-sdk/        TypeScript SDK published as @pvium/sdk
  python-sdk/      Python SDK published as pvium and imported as pvium_sdk
  go-sdk/          Go module at github.com/pvium/sdks/go-sdk
    release.yml    Go SDK release manifest used to create monorepo tags
  parity-fixtures/ Shared cross-SDK signing and payout parity fixtures
```

## Development

Run SDK checks from the package directory:

```bash
cd node-sdk && npm ci && npm run build && npm test
cd python-sdk && pip install -e '.[dev]' && pytest
cd go-sdk && go test ./...
```

The parity fixtures are shared across SDKs to keep payout hashing, signing, and
Merkle behavior consistent.

## Releases

The TypeScript SDK publishes `@pvium/sdk` to npm from
[.github/workflows/node-sdk-publish.yml](.github/workflows/node-sdk-publish.yml).

The Python SDK publishes `pvium` to PyPI from
[.github/workflows/python-sdk-publish.yml](.github/workflows/python-sdk-publish.yml).
The PyPI distribution name is `pvium`, while the Python import package remains
`pvium_sdk`.

The Go SDK is released through Git tags. Update `go_sdk.version` in
[go-sdk/release.yml](go-sdk/release.yml), merge to `main`, and the
[Go release workflow](.github/workflows/go-sdk-release.yml) will test the module
and push a tag such as `go-sdk/v0.1.0`.
