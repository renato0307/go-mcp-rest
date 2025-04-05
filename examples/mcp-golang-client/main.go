package main

import (
	"context"
	"log/slog"
	"os/exec"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"

	bookscli "github.com/renato0307/go-mcp-rest/examples/client"
)

// Define type-safe arguments

func main() {
	cmd := exec.Command("/Users/renato/Work/willful/go-mcp-rest/mcp-books")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	slog.Info("starting mcp server")
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	defer cmd.Process.Kill()

	slog.Info("creating mcp client")
	transport := stdio.NewStdioServerTransportWithIO(stdout, stdin)
	client := mcp.NewClient(transport)

	if _, err := client.Initialize(context.Background()); err != nil {
		panic(err)
	}

	slog.Info("calling mcp server")
	args := bookscli.ListBooksParams{}
	response, err := client.CallTool(context.Background(), "list-books", args)
	if err != nil {
		panic(err)
	}

	if response != nil && len(response.Content) > 0 {
		slog.Info("got a response", "result", response.Content[0].TextContent.Text)
	}
}
