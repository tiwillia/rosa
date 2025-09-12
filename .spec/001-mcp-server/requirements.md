# Requirements Specification: ROSA MCP Server Sub-command

## Problem Statement
ROSA workflows require users to understand complex step-by-step documentation and manually execute commands. AI assistants could guide users through these workflows or automate them entirely, but there's no programmatic way for AI agents to access ROSA's production-ready tooling.

## Solution Overview
Add a `rosa mcp` sub-command that runs an MCP server, exposing ROSA workflows as tools for AI agents. This enables AI assistants to guide users or autonomously complete workflows while reusing existing ROSA validation and error handling.

## Functional Requirements

### Core MCP Server Functionality
- **Sub-command Integration**: Implement `rosa mcp` as a new sub-command following existing ROSA CLI patterns
- **Blocking Foreground Process**: Server runs in the foreground as a blocking process until terminated
- **stdio Transport Only**: Support only stdio transport for security (no SSE to prevent network exposure of credentials)
- **Authentication Requirement**: Require existing OCM authentication (`rosa login`) before starting
- **Configuration Output**: Always output example MCP client configuration before starting the server
- **Signal Handling**: Gracefully handle SIGINT/SIGTERM for clean shutdown

### Initial Tool Set (Milestone 1)
- **get_cluster**: Basic tool to retrieve cluster information from OCM

### Future Tool Sets (Milestones 2-3)
- **pre_req_get_prerequisite_guide**: Markdown guide for HCP cluster prerequisites
- **create_rosa_hcp_cluster**: Tool to create ROSA HCP clusters
- **pre_req_create_account_roles**: Create account-wide IAM roles
- **pre_req_create_oidc_config**: Create OpenID Connect configuration
- **pre_req_create_operator_roles**: Create operator roles and policies
- **pre_req_setup_vpc**: Create VPC via Terraform or tag existing VPC subnets

### Command Line Interface
- **Global Flags**: Support `--color` and `--debug` flags consistent with other ROSA commands
- **No Transport Flag**: Do not implement `--transport` flag (stdio only)
- **Help Documentation**: Standard help text and examples following ROSA conventions

## Technical Requirements

### Architecture & Code Organization
- **Modular Design**: Implement MCP tools in separate packages under `pkg/mcp/` following ROSA's architecture
- **Code Reuse**: MCP tools must reuse existing functionality from other `pkg/` packages to maintain consistency
- **Protocol Compliance**: Implement MCP protocol (JSON-RPC 2.0 based) with proper initialization, capability negotiation, and tool invocation
- **Cobra Integration**: Use Cobra framework following existing ROSA command patterns

### Security Requirements
- **Local Credentials Only**: Server uses local OCM/AWS credentials, no network exposure
- **stdio Transport Limitation**: Deliberately exclude SSE transport to prevent credential exposure over network
- **Authentication Validation**: Verify OCM authentication before server startup
- **Error Handling**: Use existing ROSA reporter patterns for consistent error messaging

### Performance & Reliability
- **Graceful Shutdown**: Implement proper signal handling for clean termination
- **Resource Management**: Proper cleanup of OCM connections and AWS clients
- **Error Recovery**: Robust error handling following existing ROSA patterns

### Future Extensibility
- **Tools-Only Initially**: Start with Tools capability, design for future Resources and Prompts support
- **Workflow Expansion**: Architecture supports adding new tools for additional ROSA workflows
- **Transport Consideration**: Architecture allows future SSE support if security concerns are addressed

## Acceptance Criteria

### Milestone 1 Completion Criteria
1. **Command Registration**: `rosa mcp` command is registered and available in ROSA CLI help
2. **Authentication Check**: Command requires and validates existing OCM login before starting
3. **Configuration Output**: Server outputs valid example MCP client configuration to stdout before starting
4. **MCP Protocol**: Server implements MCP protocol initialization and capability negotiation over stdio
5. **Basic Tool**: `get_cluster` tool is functional and returns cluster information from OCM
6. **Signal Handling**: Server responds to SIGINT/SIGTERM with graceful shutdown
7. **Flag Support**: `--color` and `--debug` flags work as expected
8. **Error Handling**: Consistent error messages using ROSA reporter patterns

### Quality Gates
- **Unit Tests**: All new code covered by unit tests following ROSA Ginkgo patterns
- **Integration Tests**: MCP server startup and basic tool invocation tested
- **Code Quality**: Passes existing ROSA linting and formatting standards (`make lint`, `make fmt`)
- **Documentation**: Help text and examples match ROSA CLI conventions

## Constraints

### Security Constraints
- **stdio Transport Only**: No SSE transport implementation to prevent credential exposure
- **Local Authentication**: Must use local OCM/AWS credentials, no credential passing
- **No Network Exposure**: Server must not be accessible over network interfaces

### Technical Constraints
- **Cobra Framework**: Must use existing Cobra command structure and patterns
- **Go Version**: Must be compatible with Go 1.23.1+ as specified in go.mod
- **ROSA Patterns**: Follow existing error handling, logging, and configuration patterns
- **Backwards Compatibility**: Must not break existing ROSA CLI functionality

### Business Constraints
- **Code Reuse**: Must leverage existing ROSA packages rather than duplicating functionality
- **Consistency**: User experience must be consistent with other ROSA commands

## Dependencies

### External Dependencies
- **MCP Protocol Implementation**: Go library for MCP protocol (JSON-RPC 2.0 over stdio)
- **Existing ROSA Dependencies**: OCM SDK, AWS SDK v2, Cobra CLI framework

### Internal Dependencies
- **OCM Authentication**: Depends on existing `pkg/config` and `pkg/ocm` for authentication
- **AWS Integration**: Depends on existing `pkg/aws` for AWS operations
- **CLI Framework**: Depends on existing Cobra command structure in `cmd/rosa/main.go`

### Prerequisites
- **User Authentication**: User must have valid OCM login (`rosa login`) before starting server
- **AWS Configuration**: User must have appropriate AWS credentials configured
- **ROSA CLI**: Must be run within properly configured ROSA CLI environment

