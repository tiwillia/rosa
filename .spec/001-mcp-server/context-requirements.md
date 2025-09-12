# Context: Requirements

## Questions & Answers
Questions and provided answers to understand the problem space.

### Q1: Will the MCP server run as a blocking foreground process?
**Answer:** Yes

### Q2: Should the MCP server support both SSE and stdio transport methods?
**Answer:** No, only stdio. SSE transport poses security risks by potentially exposing user credentials over the network when the server uses local authentication.

### Q3: Will the MCP server only expose Tools (not Resources or Prompts)?
**Answer:** Yes, initially. Note however that eventually more clients are likely to support resources and prompts. When that time comes, we will likely want to take advantage of those options.

### Q4: Should the server output example configuration before starting?
**Answer:** Yes, always

### Q5: Will this be implemented as a new sub-command under the existing rosa CLI structure?
**Answer:** Yes

## Context Gathering Results
Collection of any additional detail needed to inform requirements.

### ROSA CLI Architecture Patterns
- **Command Structure**: ROSA uses Cobra framework with hierarchical commands (cmd/rosa/main.go:62-108)
- **Registration Pattern**: Commands are imported and registered in main.go init() function (cmd/rosa/main.go:78-108)
- **Command Implementation**: Two patterns observed:
  1. Simple global variable pattern: `var Cmd = &cobra.Command{}` (whoami)
  2. Factory function pattern: `func NewRosaVersionCommand() *cobra.Command` (version)
- **Runtime Pattern**: Uses `rosa.DefaultRunner()` with command runners for execution flow
- **Package Structure**: Each command has its own package under cmd/ directory

### ROSA CLI Technical Context
- **Go Version**: 1.23.1 minimum (go.mod)
- **Major Dependencies**: Cobra CLI framework, AWS SDK v2, OCM SDK
- **Error Handling**: Uses `rosa.Reporter` for consistent error messaging
- **Configuration**: Uses internal config package for OCM/AWS authentication
- **Output**: Supports multiple output formats via output package

### MCP Protocol Context  
- **Transport Methods**: stdio only (SSE excluded for security - prevents network exposure of local credentials)
- **Protocol**: JSON-RPC 2.0 based with specific MCP message types
- **Server Components**: Tools, Resources, and Prompts (though initially only Tools needed)
- **Implementation**: Requires handling initialization, capability negotiation, and tool invocation messages

### Command Integration Strategy
- Create new `cmd/mcp` package following ROSA patterns
- Register in main.go alongside other commands
- Implement blocking foreground server process
- Support stdio transport only (no SSE for security)
- Output configuration example before starting server
- Follow existing error handling and reporting patterns

## Expert Requirements Questions
Deep technical questions based on codebase understanding.

### Q6: Should the MCP server require existing OCM authentication before starting?
**Answer:** Yes, login should be required first. However, we should NOT support any transport other than stdio. This ensures that the security issues possible when the MCP server using local user credentials is exposed on the network are avoided entirely.

### Q7: Should the MCP server support the same global flags as other ROSA commands (--profile, --region, --debug)?
**Answer:** Only --color and --debug make sense for the MCP server command.

### Q8: Should transport method be selected via flag (--transport) or detected automatically?
**Answer:** --transport flag should not be added.

### Q9: Should the server gracefully handle SIGINT/SIGTERM for clean shutdown?
**Answer:** Yes

### Q10: Should initial tools be implemented in separate packages under pkg/mcp/?
**Answer:** Yes, then lets follow ROSA's existing modular arch pattern
