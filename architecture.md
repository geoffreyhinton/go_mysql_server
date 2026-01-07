# Go MySQL Server Architecture

This diagram shows the overall architecture of the Go MySQL Server project:

```
                            ┌─────────────────────────┐
                            │      Client Apps        │
                            │   (MySQL Protocol)      │
                            └─────────┬───────────────┘
                                      │ MySQL Wire Protocol
                                      ▼
                    ┌─────────────────────────────────────────────┐
                    │              SERVER LAYER                   │
                    │  ┌─────────────┐  ┌─────────────────────┐   │
                    │  │   Server    │  │      Handler        │   │
                    │  │             │  │  (MySQL Protocol    │   │
                    │  │ - Listener  │  │   Implementation)   │   │
                    │  │ - Auth      │  │                     │   │
                    │  └─────────────┘  └─────────────────────┘   │
                    └─────────────────────┬───────────────────────┘
                                          │ SQL Queries
                                          ▼
                    ┌─────────────────────────────────────────────┐
                    │               SQL ENGINE                    │
                    │  ┌─────────────┐  ┌─────────────────────┐   │
                    │  │   Engine    │  │      Catalog        │   │
                    │  │             │◄─┤                     │   │
                    │  │ - Query()   │  │ - Databases         │   │
                    │  │ - Analyzer  │  │ - Functions         │   │
                    │  │ - Catalog   │  │ - FunctionRegistry  │   │
                    │  └─────────────┘  └─────────────────────┘   │
                    └─────────────────────┬───────────────────────┘
                                          │
                        ┌─────────────────┼─────────────────┐
                        │                 │                 │
                        ▼                 ▼                 ▼
              ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
              │     PARSER      │ │   ANALYZER  │ │    EXECUTOR     │
              │                 │ │             │ │                 │
              │ ┌─────────────┐ │ │┌───────────┐│ │ ┌─────────────┐ │
              │ │    Parse    │ │ ││ Analyzer  ││ │ │    Plan     │ │
              │ │             │ │ ││           ││ │ │   Executor  │ │
              │ │ SQL Text    │ │ ││ - Rules   ││ │ │             │ │
              │ │     ↓       │ │ ││ - Validate││ │ │ RowIter()   │ │
              │ │ AST Nodes   │ │ ││ - Resolve ││ │ │             │ │
              │ └─────────────┘ │ │└───────────┘│ │ └─────────────┘ │
              └─────────────────┘ └─────────────┘ └─────────────────┘
                        │                 │                 │
                        ▼                 ▼                 ▼
              ┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
              │   SQL NODES     │ │  ANALYSIS   │ │   EXECUTION     │
              │                 │ │   RULES     │ │    PLANS        │
              │ - Expressions   │ │             │ │                 │
              │ - Plan Nodes    │ │ - Resolution│ │ - Iterators     │
              │ - Operators     │ │ - Validation│ │ - Operations    │
              └─────────────────┘ └─────────────┘ └─────────────────┘

                                          │
                                          ▼
                    ┌─────────────────────────────────────────────┐
                    │            STORAGE LAYER                    │
                    │                                             │
                    │  ┌─────────────────┐  ┌─────────────────┐   │
                    │  │   Memory Store  │  │   File System   │   │
                    │  │      (mem/)     │  │  (Future/Plugin)│   │
                    │  │                 │  │                 │   │
                    │  │ - Database      │  │ - Disk Storage  │   │
                    │  │ - Table         │  │ - Persistence   │   │
                    │  │ - Rows          │  │                 │   │
                    │  └─────────────────┘  └─────────────────┘   │
                    └─────────────────────────────────────────────┘
```

## Architecture Components:

### 1. **Server Layer** (`server/`)
- **Server**: MySQL protocol server implementation using Vitess
- **Handler**: Processes MySQL protocol messages and delegates to SQL Engine
- **Authentication**: Handles MySQL authentication protocol

### 2. **SQL Engine** (`engine.go`)
- **Engine**: Main orchestrator that coordinates parsing, analysis, and execution
- **Catalog**: Registry for databases, tables, and functions
- **Query Processing Pipeline**: Parse → Analyze → Execute

### 3. **Parser** (`sql/parse/`)
- **Parse**: Converts SQL text into Abstract Syntax Tree (AST)
- Uses Vitess SQL parser for standard SQL parsing
- Custom handling for special cases (DESCRIBE, etc.)

### 4. **Analyzer** (`sql/analyzer/`)
- **Rules Engine**: Applies transformation and optimization rules
- **Validation**: Ensures semantic correctness
- **Resolution**: Resolves table/column references
- **Optimization**: Query optimization transformations

### 5. **SQL Components** (`sql/`)
- **Core Types**: Basic SQL types (Schema, Row, Node, etc.)
- **Expressions**: SQL expressions and operators
- **Plan Nodes**: Execution plan node implementations
- **Function Registry**: Built-in and user-defined functions

### 6. **Execution Plans** (`sql/plan/`)
- **Plan Nodes**: Physical operators (Filter, Project, Join, etc.)
- **Iterators**: Row-by-row execution model
- **Operations**: Specific SQL operations (INSERT, SELECT, etc.)

### 7. **Storage Layer**
- **Memory Store** (`mem/`): In-memory database implementation
- **Pluggable**: Interface allows custom storage backends
- **Tables & Databases**: Storage abstractions

## Data Flow:

1. **Client Request**: MySQL client sends SQL query
2. **Protocol Handling**: Server/Handler processes MySQL protocol
3. **Parsing**: SQL text → AST nodes
4. **Analysis**: Apply rules, validate, optimize
5. **Execution**: Generate row iterator, process data
6. **Storage**: Read/write from storage backend
7. **Response**: Return results via MySQL protocol

## Key Design Patterns:

- **Pipeline Architecture**: Clear separation of Parse → Analyze → Execute
- **Visitor Pattern**: Tree traversal for analysis and execution
- **Iterator Pattern**: Row-by-row processing for memory efficiency
- **Plugin Architecture**: Pluggable storage backends
- **Rule Engine**: Extensible analysis and optimization rules