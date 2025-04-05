# go-mcp-rest

A toolkit to generate MCP (Model-Controller-Provider) server and client code from an OpenAPI specification.

## Installation

```bash
go install github.com/renato0307/go-mcp-rest/cmd/mcp-rest-client-gen@latest
go install github.com/renato0307/go-mcp-rest/cmd/mcp-rest-server-gen@latest
```

## Workflow

The recommended workflow is:

1. First, generate the client stubs using `mcp-rest-client-gen`
2. Then, generate the server code using `mcp-rest-server-gen`

## Client Generation

```bash
mcp-rest-client-gen --spec=STRING [flags]
```

Generates Go client code from an OpenAPI specification using oapi-codegen.

For a complete list of available flags and options:

```bash
mcp-rest-client-gen --help
```

For a complete list of available flags and their default values, use the help command.

### Client Generation Examples

```bash
# Generate client with default settings
mcp-rest-client-gen --spec=https://example.com/api/openapi.json

```

## Server Generation

```bash
mcp-rest-server-gen [flags]
```

Generates a MCP server from an OpenAPI specification, creating the necessary code structure following the Model-Controller-Provider pattern.

For a complete list of available flags and options:

```bash
mcp-rest-server-gen --help
```

### Server Generation Examples

```bash
# Generate server code from a remote OpenAPI specification
mcp-rest-server-gen --spec=https://example.com/api/openapi.json
```

## Building the Server

After generating the server code, you can build the server using the following command:

```bash
go build -o mcp-server ./path/to/generated/server
```

By default it will use the `generated` directory in the current working directory. 

```
go build -o mcp-server ./generated
```

## Authentication

The generated server will use the username and password from the environment variables for API authentication. By default, these are `API_USERNAME` and `API_PASSWORD`, but can be customized using the appropriate flags.

## Claude Desktop Integration

To configure Claude Desktop to use your MCP server:

1. Locate and modify the Claude Desktop settings file:
   - On Windows: `%APPDATA%\Claude Desktop\settings.json`
   - On macOS: `~/Library/Application Support/Claude Desktop/settings.json`
   - On Linux: `~/.config/Claude Desktop/settings.json`

2. Add or modify the MCP server configuration in the settings file:
   ```json
   {
     "mcpServers": {
       "put-here-the-mcp-server-name": {
         "command": "/path/to/your/mcp-server",
         "args": [],
         "env": {
           "API_USERNAME": "your-username",
           "API_PASSWORD": "your-password"
         }
       }
     }
   }
   ```

   Replace:
    - `put-here-the-mcp-server-name` with a name for your MCP server
   - `/path/to/your/mcp-server` with the full path to your compiled MCP server executable
   - `your-username` with your desired username
   - `your-password` with your desired password
   - Adjust environment variable names if you used custom ones with `--username-env` and `--password-env`

3. Save the file and restart Claude Desktop

Claude Desktop will now start your MCP server as a subprocess when needed and communicate with it directly.

## License

[MIT](LICENSE)