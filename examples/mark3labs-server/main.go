package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"

	"github.com/renato0307/go-mcp-rest/examples/client"
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"Demo 🚀",
		"1.0.0",
	)

	// Add tool
	tool := mcp.NewTool("list-books",
		mcp.WithDescription("List the books filtered by name"),
		mcp.WithString("Filter",
			mcp.Description("Filter books by name"),
			mcp.Title("Filter"),
		),
	)

	// Add tool handler
	s.AddTool(tool, listBooksHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func listBooksHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filter, ok := request.Params.Arguments["Filter"].(string)
	if !ok {
		return nil, errors.New("filter must be a string")
	}
	arguments := client.ListBooksParams{Filter: &filter}

	basicAuth, err := securityprovider.NewSecurityProviderBasicAuth(os.Getenv("API_USERNAME"), os.Getenv("API_PASSWORD"))
	if err != nil {
		log.Fatal(err)
	}

	restClient, err := client.NewClientWithResponses("https://eng-test-us-01-dev.outsystems.app/MCPBackend/rest/Backend", client.WithRequestEditorFn(basicAuth.Intercept))
	if err != nil {
		panic(err)
	}

	resp, err := restClient.ListBooksWithResponse(context.TODO(), &arguments)
	if err != nil {
		return nil, fmt.Errorf("error calling list books: %v", err)
	}
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("error on list books: %s", resp.Status())
	}

	return mcp.NewToolResultText(string(resp.Body)), nil
}
