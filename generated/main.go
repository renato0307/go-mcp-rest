package main

import (
	"context"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"github.com/renato0307/go-mcp-rest/generated/api"
	"log"
	"log/slog"
)

func main() {
	var cli = struct {
		Host     string `default:"https://eng-test-us-01-dev.outsystems.app/MCPBackend/rest/Backend" help:"API server host"`
		Username string `env:"API_USERNAME" help:"API username"`
		Password string `env:"API_PASSWORD" help:"API password"`
	}{}
	kong.Parse(&cli)
	done := make(chan struct{})
	basicAuth, err := securityprovider.NewSecurityProviderBasicAuth(cli.Username, cli.Password)
	if err != nil {
		log.Fatal(err)
	}
	restClient, err := api.NewClientWithResponses(cli.Host, api.WithRequestEditorFn(basicAuth.Intercept))
	if err != nil {
		panic(err)
	}
	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())
	err = server.RegisterTool("AddBook", "Adds a new book", func(arguments api.AddBookJSONRequestBody) (*mcp_golang.ToolResponse, error) {
		resp, err := restClient.AddBookWithResponse(context.TODO(), arguments)
		if err != nil {
			return nil, fmt.Errorf("error calling AddBook: %v", err)
		}
		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf("error on AddBook: %s", resp.Status())
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(string(resp.Body))), nil
	})
	if err != nil {
		panic(err)
	}
	err = server.RegisterTool("ListBooks", "Lists books filtering by name.", func(arguments api.ListBooksParams) (*mcp_golang.ToolResponse, error) {
		resp, err := restClient.ListBooksWithResponse(context.TODO(), &arguments)
		if err != nil {
			return nil, fmt.Errorf("error calling ListBooks: %v", err)
		}
		if resp.StatusCode() != 200 {
			return nil, fmt.Errorf("error on ListBooks: %s", resp.Status())
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
