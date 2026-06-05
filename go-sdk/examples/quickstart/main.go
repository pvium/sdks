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

	fmt.Printf("success=%v count=%d\n", invoices.Meta.Success, len(invoices.Data))
}
