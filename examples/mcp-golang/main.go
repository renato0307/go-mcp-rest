package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"

	"github.com/renato0307/go-mcp-rest/examples/client"
)

func main() {
	done := make(chan struct{})
	basicAuth, err := securityprovider.NewSecurityProviderBasicAuth(os.Getenv("API_USERNAME"), os.Getenv("API_PASSWORD"))
	if err != nil {
		log.Fatal(err)
	}

	restClient, err := client.NewClientWithResponses("https://eng-test-us-01-dev.outsystems.app/MCPBackend/rest/Backend", client.WithRequestEditorFn(basicAuth.Intercept))
	if err != nil {
		panic(err)
	}

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	err = server.RegisterTool("list-books", "List the books filtered by name", func(arguments client.ListBooksParams) (*mcp_golang.ToolResponse, error) {
		resp, err := restClient.ListBooksWithResponse(context.TODO(), &arguments)
		if err != nil {
			return nil, fmt.Errorf("error calling list books: %v", err)
		}
		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf("error on list books: %s", resp.Status())
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(resp.Body))), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	slog.Info("Server started")

	<-done
}
