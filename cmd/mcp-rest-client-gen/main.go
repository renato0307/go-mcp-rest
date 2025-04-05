package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
)

// CLI defines the command-line interface structure
type CLI struct {
	Spec           string `name:"spec" help:"Path or URL to the OpenAPI spec" required:""`
	OutputDir      string `name:"output-dir" help:"Output directory for the generated client code" default:"./generated/api"`
	Filename       string `name:"filename" help:"Name of the generated file" default:"client.go"`
	Package        string `name:"package" help:"Package name for the generated code" default:"api"`
	GenerateTypes  bool   `name:"generate-types" help:"Generate type definitions" default:"true"`
	GenerateClient bool   `name:"generate-client" help:"Generate client code" default:"true"`
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli,
		kong.Name("mcp-rest-client-gen"),
		kong.Description("Generate Go client code from OpenAPI spec using oapi-codegen"))

	// Get the spec content
	specContent, err := getSpecContent(cli.Spec)
	if err != nil {
		ctx.FatalIfErrorf(err, "Error getting spec content")
	}

	// Save the spec content to a temporary file
	tempDir, err := os.MkdirTemp("", "oapi-codegen")
	if err != nil {
		ctx.FatalIfErrorf(err, "Error creating temp directory")
	}
	defer os.RemoveAll(tempDir)

	tempSpecPath := filepath.Join(tempDir, "spec.yaml")
	if err := os.WriteFile(tempSpecPath, specContent, 0644); err != nil {
		ctx.FatalIfErrorf(err, "Error writing spec to temp file")
	}

	// Create config file for oapi-codegen
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := createConfigContent(cli.Package, cli.GenerateTypes, cli.GenerateClient)
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		ctx.FatalIfErrorf(err, "Error writing config file")
	}

	// Check if oapi-codegen is installed
	if _, err := exec.LookPath("oapi-codegen"); err != nil {
		log.Println("oapi-codegen not found. Installing...")
		cmd := exec.Command("go", "install", "github.com/kin-openapi/oapi-codegen/v2/cmd/oapi-codegen@latest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			ctx.FatalIfErrorf(err, "Error installing oapi-codegen")
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(cli.OutputDir, 0755); err != nil {
		ctx.FatalIfErrorf(err, "Error creating output directory")
	}

	// Try the direct command approach first
	log.Println("Generating client code...")

	// First check if we got a new enough version of oapi-codegen that supports the "-config" flag
	versionCmd := exec.Command("oapi-codegen", "--version")
	versionOutput, _ := versionCmd.CombinedOutput()
	log.Printf("oapi-codegen version: %s\n", string(versionOutput))

	// Create a complete command with all parameters directly
	var cmd *exec.Cmd
	if cli.GenerateClient && cli.GenerateTypes {
		cmd = exec.Command("oapi-codegen",
			"-package", cli.Package,
			"-generate", "types,client",
			tempSpecPath)
	} else if cli.GenerateClient {
		cmd = exec.Command("oapi-codegen",
			"-package", cli.Package,
			"-generate", "client",
			tempSpecPath)
	} else if cli.GenerateTypes {
		cmd = exec.Command("oapi-codegen",
			"-package", cli.Package,
			"-generate", "types",
			tempSpecPath)
	} else {
		ctx.FatalIfErrorf(fmt.Errorf("invalid options"), "At least one of generate-types or generate-client must be true")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error output from oapi-codegen:")
		log.Println(string(output))
		// Try alternative approach with config file
		configCmd := exec.Command("oapi-codegen", "-config", configPath, tempSpecPath)
		output, err = configCmd.CombinedOutput()
		if err != nil {
			log.Println("Config file approach also failed:")
			log.Println(string(output))
			log.Printf("Config file content:\n%s\n", configContent)
			ctx.FatalIfErrorf(err, "Error running oapi-codegen")
		}
	}

	// Check if output is empty
	if len(output) == 0 {
		log.Println("Warning: oapi-codegen produced empty output")
		ctx.FatalIfErrorf(fmt.Errorf("empty output"), "oapi-codegen generated empty output")
	}

	// Write the generated code to the output file
	outputFilePath := filepath.Join(cli.OutputDir, cli.Filename)
	if err := os.WriteFile(outputFilePath, output, 0644); err != nil {
		ctx.FatalIfErrorf(err, "Error writing output file")
	}

	log.Printf("Successfully generated client code at %s\n", outputFilePath)
}

// getSpecContent retrieves the OpenAPI spec content from a URL or file path
func getSpecContent(specPath string) ([]byte, error) {
	// Check if the spec path is a URL
	if strings.HasPrefix(specPath, "http://") || strings.HasPrefix(specPath, "https://") {
		// Fetch the spec from the URL
		resp, err := http.Get(specPath)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch spec from URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("received non-OK response: %s", resp.Status)
		}

		return io.ReadAll(resp.Body)
	}

	// Otherwise, read from the file
	return os.ReadFile(specPath)
}

// createConfigContent creates the configuration content for oapi-codegen
func createConfigContent(packageName string, generateTypes, generateClient bool) string {
	// Start with package name
	config := fmt.Sprintf("package: %s\n", packageName)

	// Add output target
	config += "output: stdout\n"

	// Add generate options
	config += "generate:\n"

	if generateTypes {
		config += "  models: true\n"
	} else {
		config += "  models: false\n"
	}

	if generateClient {
		config += "  client: true\n"
	} else {
		config += "  client: false\n"
	}

	// Set other generators to false
	config += "  echo-server: false\n"
	config += "  chi-server: false\n"
	config += "  fiber-server: false\n"
	config += "  gin-server: false\n"
	config += "  strict-server: false\n"

	return config
}
