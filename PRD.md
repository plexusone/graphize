# Graphize - Product Requirements Document

## Overview

Graphize is an LLM-powered CLI tool that transforms Go codebases into queryable knowledge graphs. It combines deterministic AST extraction with optional LLM semantic analysis to create rich, navigable representations of code architecture.

## Problem Statement

Understanding large codebases is hard. Developers need to:
- Understand relationships between components
- Find unexpected dependencies
- Navigate unfamiliar code quickly
- Share architectural knowledge with AI agents

Existing tools either:
- Require manual documentation (gets stale)
- Only show explicit dependencies (miss semantic relationships)
- Output formats that don't integrate with AI workflows

## Solution

Graphize provides a two-step extraction pipeline:

1. **Deterministic AST extraction** - Fast, reproducible, always available
2. **LLM semantic extraction** - Optional, adds inferred relationships and rationale

Output is stored in git-friendly format (one file per entity) and can be exported to multiple formats including interactive HTML, TOON (for AI agents), and graph databases.

## Target Users

1. **Developers** exploring unfamiliar codebases
2. **AI Agents** (Claude, Codex) needing codebase context
3. **Architects** documenting system design
4. **Teams** onboarding new members

## User Stories

### Core Workflow
- As a developer, I want to extract a graph from my Go codebase so I can understand its architecture
- As a developer, I want to query the graph for specific nodes and their relationships
- As a developer, I want to visualize the graph in my browser
- As a developer, I want the graph data to be git-friendly so I can track changes

### AI Agent Integration
- As an AI agent, I want to read a compact graph summary (TOON format) for context
- As an AI agent, I want to regenerate the full graph locally when needed
- As an AI agent, I want to query specific paths and relationships

### LLM Enhancement
- As a developer, I want LLM to infer relationships not visible in AST
- As a developer, I want confidence scores on inferred relationships
- As a developer, I want to see surprising/unexpected connections
- As a developer, I want suggested questions the graph can answer

## Feature Comparison: Graphize vs Graphify

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     GRAPHIZE vs GRAPHIFY                                 │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  GRAPHIZE ADVANTAGES (Go implementation)                                │
│  ├── Multi-repo support with direct filepaths                           │
│  ├── Git commit/branch tracking per source                              │
│  ├── Git-friendly storage (one file per entity)                         │
│  ├── TOON format for agent-friendly output                              │
│  └── Cytoscape.js visualization                                         │
│                                                                          │
│  GRAPHIFY ADVANTAGES (Python implementation)                            │
│  ├── 20 language support via tree-sitter                                │
│  ├── LLM semantic extraction with subagents                             │
│  ├── Community detection (Leiden/Louvain)                               │
│  ├── Hyperedges for group relationships                                 │
│  ├── Obsidian/Neo4j export                                              │
│  └── Watch mode and git hooks                                           │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

| Feature | Graphify | Graphize | Status |
|---------|----------|----------|--------|
| **Source Tracking** | Single directory | Multi-repo with commit hashes | ✅ Better |
| **Git Currency** | None | Tracks commit/branch per repo | ✅ Better |
| **Storage** | Single graph.json | One file per entity | ✅ Better |
| **AST Extraction** | tree-sitter (20 langs) | Go only | ✅ Done |
| **LLM Semantic Extraction** | ✅ Subagents | ❌ Not implemented | 🎯 Priority |
| **Per-file Caching** | ✅ SHA256 | ❌ Not implemented | 🎯 Priority |
| **Community Detection** | ✅ Leiden/Louvain | ❌ Not implemented | Medium |
| **God Nodes Analysis** | ✅ Full | Partial (summary) | Medium |
| **Surprising Connections** | ✅ | ❌ | Medium |
| **Suggested Questions** | ✅ | ❌ | Low |
| **Hyperedges** | ✅ | ❌ | Low |
| **HTML Visualization** | ✅ vis.js | ✅ cytoscape.js | ✅ Done |
| **TOON Export** | ❌ | ✅ | ✅ Done |
| **GRAPH_REPORT.md** | ✅ | ❌ | Medium |
| **Obsidian Export** | ✅ | ❌ | Low |
| **Neo4j Export** | ✅ | ❌ | Low |
| **Watch Mode** | ✅ | ❌ | Low |
| **MCP Server** | ✅ | ❌ | Medium |
| **Git Hooks** | ✅ | ❌ | Low |

## Requirements

### P0 - Must Have
1. Two-step extraction (AST + optional LLM)
2. Confidence levels on edges (EXTRACTED, INFERRED, AMBIGUOUS)
3. Per-file caching to avoid redundant LLM calls
4. TOON export for agent consumption
5. HTML visualization

### P1 - Should Have
1. Community detection
2. GRAPH_REPORT.md generation
3. God nodes and surprising connections analysis
4. MCP server for agent integration

### P2 - Nice to Have
1. Obsidian/Neo4j export
2. Watch mode
3. Git hooks
4. Multi-language support

## Success Metrics

1. Extract 20K+ node graph in <30 seconds (AST only)
2. LLM enhancement adds <5 minutes for typical codebase
3. TOON output <500KB for typical codebase (gzipped)
4. HTML visualization loads in <3 seconds for 20K nodes

## Non-Goals

1. Real-time graph updates (batch processing is fine)
2. Multi-user collaboration features
3. Cloud hosting / SaaS offering
4. Language parity with graphify (Go-first)
