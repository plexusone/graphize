# MCP Server

Graphize includes a Model Context Protocol (MCP) server for integration with AI assistants like Claude Desktop and Claude Code.

## Overview

The MCP server exposes graph query tools that AI agents can use to explore and understand your codebase.

## Starting the Server

```bash
graphize serve -g /path/to/graph
```

The server runs over stdio and can be used with any MCP-compatible client.

## Claude Desktop Configuration

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "graphize": {
      "command": "graphize",
      "args": ["serve", "-g", "/path/to/your/project/.graphize"]
    }
  }
}
```

## Claude Code Configuration

Add to your project's `.claude/settings.json`:

```json
{
  "mcpServers": {
    "graphize": {
      "command": "graphize",
      "args": ["serve", "-g", ".graphize"]
    }
  }
}
```

## Available Tools

### query_graph

Search and traverse the graph from matching nodes.

**Input:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `query` | string | Search terms or node label |
| `mode` | string | Traversal mode: `bfs` or `dfs` (default: `bfs`) |
| `depth` | int | Traversal depth 1-6 (default: 2) |
| `max_results` | int | Maximum results (default: 20) |

**Example:**

```
Query the graph for "handleRequest" with depth 3
```

### get_node

Get full details for a specific node.

**Input:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Node ID or label |

**Output:**

- Node ID, label, type
- Attributes
- In-degree and out-degree
- Community membership

### get_neighbors

Get all direct neighbors of a node.

**Input:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | string | Node ID |
| `direction` | string | `in`, `out`, or `both` (default: `both`) |

**Output:**

- List of neighbor nodes
- Edge types connecting them
- Direction of each edge

### get_community

Get all nodes in a specific community.

**Input:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | int | Community ID |

**Output:**

- List of member nodes
- Community size
- Community label

### graph_summary

Get overall graph statistics.

**Input:** None

**Output:**

- Total nodes and edges
- Node types breakdown
- Edge types breakdown
- Number of communities
- God nodes (most connected)
- Edge confidence breakdown
- Suggested questions

## Example Conversations

### Exploring Architecture

> **You:** What are the most connected functions in this codebase?

> **Claude:** Let me check the graph summary.
>
> *Uses graph_summary tool*
>
> The most connected functions (god nodes) are:
> 1. `HandleRequest` - 45 connections (central request handler)
> 2. `NewService` - 32 connections (service factory)
> 3. `ValidateInput` - 28 connections (used by most handlers)

### Finding Dependencies

> **You:** What does the `ProcessOrder` function depend on?

> **Claude:** Let me trace the dependencies.
>
> *Uses query_graph with query="ProcessOrder", mode="bfs", depth=2*
>
> `ProcessOrder` depends on:
> - `ValidateOrder` (calls)
> - `InventoryService.CheckStock` (calls)
> - `PaymentService.Process` (calls)
> - `NotificationService.Send` (calls)

### Understanding Communities

> **You:** What functions are in the same community as `AuthMiddleware`?

> **Claude:** Let me find that community.
>
> *Uses get_node to find AuthMiddleware's community*
> *Uses get_community to get members*
>
> `AuthMiddleware` is in Community 3 (Authentication) with:
> - `ValidateToken`
> - `RefreshToken`
> - `HashPassword`
> - `CheckPermissions`

## Best Practices

### 1. Keep Graphs Updated

Run `graphize analyze` after significant code changes to keep the MCP server's data current.

### 2. Use Semantic Extraction

LLM-enhanced graphs provide richer relationships:

```bash
graphize analyze
# Run semantic extraction
graphize merge -i agents/graph/semantic-edges.json
```

### 3. Pre-generate Reports

Have Claude read the report first for context:

```bash
graphize report -o agents/graph/GRAPH_REPORT.md
```

### 4. Scope Large Graphs

For very large codebases, consider:

- Multiple smaller graphs per package
- Pre-computed community summaries
- Filtered graphs (e.g., exclude tests)

## Troubleshooting

### Server Not Starting

Check that the graph path exists and contains data:

```bash
ls -la /path/to/.graphize/nodes/
```

### No Results Returned

Ensure the graph has been analyzed:

```bash
graphize analyze
graphize query  # Verify graph has content
```

### Slow Responses

For large graphs, the initial load may take a few seconds. Subsequent queries are faster.
