# Initial Requirements for Embedded ROSA MCP

## Summary

ROSA MCP sever embedded in ROSA CLI.

Users run `rosa mcp` and the mcp server runs in the foreground. The MCP server exposes common workflows - not every
single command-line option. Its meant to provide a prompt to an LLM guide the LLM through performing various common
workflows using ROSA CLI internal tooling exposed through the MCP. It can then provide details on how the user can
follow up and troubleshoot using the ROSA CLI.

## Purpose

The ROSA CLI provides a robust interface to OpenShift Cluster Manager tailored specifically for ROSA cluster
installation and maintenance. However, many workflows require intimate knowledge of ROSA requirements, internals, and
details. These workflows are currently described in details in various documents, often with step-by-step instructions
and example ROSA CLI commands for users to run

The ROSA MCP server enables an LLM to either guide users through usage of the ROSA CLI tooling to complete these complex
workflows or the ability for an LLM Agent to complete the workflows using tooling exposed by the MCP server.

### Why embed in the ROSA CLI?

The ROSA CLI already has many packages and libraries that provide excellent and production-ready validations and error
handling on top of the existing OCM API to complete ROSA-specific tasks. Embedding the MCP server in the CLI allows the
MCP server to re-use all of those existing tools.

Embedding also enables users to run the MCP locally, ensuring access to privledged credentials and tools (OCM, AWS,
Terraform, etc) to enable automated completion of previously manual tasks - primarily in the pre-resiquite setup
workflow.

Additionally, embedding the MCP server ensures that the workflow guidance is maintained alongside the tooling.

## How

### ROSA CLI Sub-command

The ROSA CLI today is a large collection of sub-commands:
```
$ rosa --help
...
Usage:
  rosa [command]

Available Commands:
  completion  Generates completion scripts
  config      get or set configuration variables
  create      Create a resource from stdin
...
```

A new sub-command `mcp` will be added that starts an MCP server. This sub-command will support its own collection of
necessary flags, such as which transport to use.

### Tools

The MCP server will only expose tools. The vast majority of MCP clients in use only support Tools, even though some of
the tools we will expose would make more sense as Resources or Prompts.

### Workflows

Possible workflows:
- ROSA HCP cluster pre-resiquite setup
- ROSA HCP cluster installation
- IDP Setup (supporting htpasswd, github, and oauth2)
- Cluster Scaling
- Upgrade scheduling

#### ROSA HCP Installation pre-requisite setup

The most important workflow is "ROSA HCP Installation pre-requisite setup". This workflow will guide the LLM through
helping a user peform the necessary steps to set up all resources required for a ROSA HCP cluster installation and guide
the LLM to collect the right information from those steps to provide to the ROSA HCP cluster installation tool.

## Potential Milestones

**Milestone 1**:
- `rosa mcp` sub-command supported
- `rosa mcp` sub-command runs an MCP server in the foreground exposing a single basic tool: "get_cluster"
- `rosa mcp` supports transport selection command-line option(s) with values SSE or stdio
- `rosa mcp` sub-command outputs example mcp server configuration to add to LLM mcp client settings before starting

**Milestone 2**:
- `rosa mcp` sub-command exposes a tool "pre_req_get_prerequisite_guide" providing a markdown prompt guiding an
  LLM through a workflow to set up pre-reqs
- `rosa mcp` sub-command exposes a tool "create_rosa_hcp_cluster" to create a ROSA HCP cluster

**Milestone 3**:
- `rosa mcp` sub-command exposes a tool "pre_req_create_account_wide_roles" to create the account wide roles
- `rosa mcp` sub-command exposes a tool "pre_req_create_oidc_config" to create the OpenID Connect (OIDC) Configuration
- `rosa mcp` sub-command exposes a tool "pre_req_create_operator_roles_policies" to create the Operator Roles and
  Policies
