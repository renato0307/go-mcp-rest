package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/dave/jennifer/jen"
	"github.com/getkin/kin-openapi/openapi3"
)

// CLI represents the command-line interface configuration
var CLI struct {
	Spec          string `help:"Path or URL to the OpenAPI specification" default:"https://converter.swagger.io/api/convert?url=https://eng-test-us-01-dev.outsystems.app/MCPBackend/rest/Backend/swagger.json"`
	Output        string `help:"Output file for the generated code" default:"./generated/main.go"`
	Package       string `help:"Package name for the generated code" default:"main"`
	ClientPackage string `help:"Name of the client package" default:"api"`
	ClientImport  string `help:"Import path for the client package" default:"github.com/renato0307/go-mcp-rest/generated/api"`
	ServerURL     string `help:"URL of the API server" default:"https://eng-test-us-01-dev.outsystems.app/MCPBackend/rest/Backend"`
	UsernameEnv   string `help:"Environment variable name for username" default:"API_USERNAME"`
	PasswordEnv   string `help:"Environment variable name for password" default:"API_PASSWORD"`
}

// OperationInfo holds information about an API operation
type OperationInfo struct {
	ID             string
	Summary        string
	Description    string
	ParameterType  string
	HasRequestBody bool
}

func main() {
	ctx := kong.Parse(&CLI, kong.Name("mcp-rest-server-gen"), kong.Description("Generate a MCP server from an OpenAPI spec"))

	// If output path is empty, generate it from the URL
	if CLI.Output == "" {
		// Extract application name from server URL
		appName := extractAppNameFromURL(CLI.ServerURL)
		outputDir := filepath.Join("cmd", strings.ToLower(appName))

		// Create the directory if it doesn't exist
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			ctx.FatalIfErrorf(fmt.Errorf("error creating output directory: %w", err))
		}

		CLI.Output = filepath.Join(outputDir, "main.go")
		fmt.Printf("Output file not specified, using: %s\n", CLI.Output)
	}

	// Generate MCP server code
	if err := generateMCPServer(); err != nil {
		ctx.FatalIfErrorf(err)
	}

	fmt.Printf("MCP server generated successfully: %s\n", CLI.Output)
}

// extractAppNameFromURL extracts the first path segment after the host from a URL
func extractAppNameFromURL(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		// Default to "app" if URL parsing fails
		return "app"
	}

	// Split the path by slashes and get the first non-empty segment
	pathParts := strings.Split(parsedURL.Path, "/")
	for _, part := range pathParts {
		if part != "" {
			return part
		}
	}

	// Default to "app" if no path segments found
	return "app"
}

// loadOpenAPISpec loads an OpenAPI specification from either a file or URL
func loadOpenAPISpec(specPath string) (*openapi3.T, error) {
	var doc *openapi3.T
	loader := openapi3.NewLoader()

	// Check if the path is a URL
	parsedURL, parseErr := url.Parse(specPath)
	if parseErr == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") {
		// It's a URL, load from URL
		fmt.Printf("Loading OpenAPI spec from URL: %s\n", specPath)

		// Fetch the content
		resp, err := http.Get(specPath)
		if err != nil {
			return nil, fmt.Errorf("error fetching from URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
		}

		// Read the content
		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		// Parse the document
		// LoadFromData automatically handles both JSON and YAML formats
		doc, err = loader.LoadFromData(content)
		if err != nil {
			return nil, fmt.Errorf("error parsing OpenAPI spec: %w", err)
		}
	} else {
		// It's a file path, load from file
		fmt.Printf("Loading OpenAPI spec from file: %s\n", specPath)
		var err error
		doc, err = loader.LoadFromFile(specPath)
		if err != nil {
			return nil, fmt.Errorf("error loading OpenAPI spec from file: %w", err)
		}
	}

	// Validate the spec
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	return doc, nil
}

func generateMCPServer() error {
	// Load and parse OpenAPI spec
	doc, err := loadOpenAPISpec(CLI.Spec)
	if err != nil {
		return err
	}

	// Extract operations from the spec
	operations := make(map[string]OperationInfo)

	// Correctly iterate through paths
	for path := range doc.Paths.Map() {
		pathItem := doc.Paths.Find(path)
		if pathItem == nil {
			continue
		}

		// Process all operations for this path (GET, POST, etc.)
		processOperation(path, "GET", pathItem.Get, operations)
		processOperation(path, "POST", pathItem.Post, operations)
		processOperation(path, "PUT", pathItem.Put, operations)
		processOperation(path, "DELETE", pathItem.Delete, operations)
		processOperation(path, "PATCH", pathItem.Patch, operations)
		processOperation(path, "HEAD", pathItem.Head, operations)
		processOperation(path, "OPTIONS", pathItem.Options, operations)
	}

	if len(operations) == 0 {
		return fmt.Errorf("no valid operations found in the OpenAPI spec")
	}

	fmt.Printf("Found %d operations in the OpenAPI spec\n", len(operations))

	// Generate code using jennifer
	f := jen.NewFile(CLI.Package)

	// Add imports
	f.ImportName("context", "context")
	f.ImportName("fmt", "fmt")
	f.ImportName("log", "log")
	f.ImportName("log/slog", "slog")
	f.ImportName("os", "os")
	f.ImportName("github.com/metoro-io/mcp-golang", "mcp_golang")
	f.ImportName("github.com/metoro-io/mcp-golang/transport/stdio", "stdio")
	f.ImportName("github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider", "securityprovider")
	f.ImportName("github.com/alecthomas/kong", "kong")
	f.ImportName(CLI.ClientImport, CLI.ClientPackage)

	// Define the main function properly
	mainBody := []jen.Code{
		// Define flags
		jen.Var().Id("cli").Op("=").Struct(
			jen.Id("Host").String().Tag(map[string]string{"help": "API server host", "default": CLI.ServerURL}),
			jen.Id("Username").String().Tag(map[string]string{"help": "API username", "env": CLI.UsernameEnv}),
			jen.Id("Password").String().Tag(map[string]string{"help": "API password", "env": CLI.PasswordEnv}),
		).Op("{}"),

		// Parse flags
		jen.Qual("github.com/alecthomas/kong", "Parse").Call(jen.Op("&").Id("cli")),

		// Create a done channel
		jen.Id("done").Op(":=").Make(jen.Chan().Struct()),

		// Setup basic auth
		jen.List(jen.Id("basicAuth"), jen.Err()).Op(":=").Qual("github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider", "NewSecurityProviderBasicAuth").Call(
			jen.Id("cli").Dot("Username"),
			jen.Id("cli").Dot("Password"),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Qual("log", "Fatal").Call(jen.Err()),
		),

		// Create REST client
		jen.List(jen.Id("restClient"), jen.Err()).Op(":=").Qual(CLI.ClientImport, "NewClientWithResponses").Call(
			jen.Id("cli").Dot("Host"),
			jen.Qual(CLI.ClientImport, "WithRequestEditorFn").Call(jen.Id("basicAuth").Dot("Intercept")),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Panic(jen.Err()),
		),

		// Create server
		jen.Id("server").Op(":=").Qual("github.com/metoro-io/mcp-golang", "NewServer").Call(
			jen.Qual("github.com/metoro-io/mcp-golang/transport/stdio", "NewStdioServerTransport").Call(),
		),
	}

	// Add tools registration for each operation
	for _, op := range operations {
		paramExpr := jen.Id("arguments")
		if !op.HasRequestBody {
			paramExpr = jen.Op("&").Id("arguments")
		}

		mainBody = append(mainBody,
			jen.Err().Op("=").Id("server").Dot("RegisterTool").Call(
				jen.Lit(op.ID),
				jen.Lit(op.Description),
				jen.Func().Params(
					jen.Id("arguments").Qual(CLI.ClientImport, op.ParameterType),
				).Params(
					jen.Op("*").Qual("github.com/metoro-io/mcp-golang", "ToolResponse"),
					jen.Error(),
				).Block(
					jen.List(jen.Id("resp"), jen.Err()).Op(":=").Id("restClient").Dot(op.ID+"WithResponse").Call(
						jen.Qual("context", "TODO").Call(),
						paramExpr,
					),
					jen.If(jen.Err().Op("!=").Nil()).Block(
						jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("error calling "+op.ID+": %v"), jen.Err())),
					),
					jen.If(jen.Id("resp").Dot("StatusCode").Call().Op("!=").Lit(200)).Block(
						jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("error on "+op.ID+": %s"), jen.Id("resp").Dot("Status").Call())),
					),
					jen.Return(
						jen.Qual("github.com/metoro-io/mcp-golang", "NewToolResponse").Call(
							jen.Qual("github.com/metoro-io/mcp-golang", "NewTextContent").Call(
								jen.String().Call(jen.Id("resp").Dot("Body")),
							),
						),
						jen.Nil(),
					),
				),
			),
			jen.If(jen.Err().Op("!=").Nil()).Block(
				jen.Panic(jen.Err()),
			),
		)
	}

	// Add server start and wait for done
	mainBody = append(mainBody,
		jen.Err().Op("=").Id("server").Dot("Serve").Call(),
		jen.If(jen.Err().Op("!=").Nil()).Block(
			jen.Panic(jen.Err()),
		),

		jen.Qual("log/slog", "Info").Call(jen.Lit("Server started")),
		jen.Op("<-").Id("done"),
	)

	// Add the proper main function to the file
	f.Func().Id("main").Params().Block(mainBody...)

	// Save the file
	return f.Save(CLI.Output)
}

// processOperation handles an individual operation within a path
func processOperation(path, method string, operation *openapi3.Operation, operations map[string]OperationInfo) {
	if operation == nil || operation.OperationID == "" {
		return
	}

	paramType := fmt.Sprintf("%sParams", operation.OperationID)
	hasRequestBody := false

	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		hasRequestBody = true
		paramType = fmt.Sprintf("%sJSONRequestBody", operation.OperationID)
	}

	summary := operation.Summary
	if summary == "" {
		summary = fmt.Sprintf("%s %s", method, path)
	}

	description := operation.Description
	if description == "" {
		description = summary
	}

	operations[operation.OperationID] = OperationInfo{
		ID:             operation.OperationID,
		Summary:        summary,
		Description:    description,
		ParameterType:  paramType,
		HasRequestBody: hasRequestBody,
	}
}
