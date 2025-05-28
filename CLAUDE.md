# CLAUDE.md

This repository is a Go implementation of the Schematic rules engine. It is used in both the Schematic public Go SDK (github.com/schematichq/schematic-go) and the private Schematic API. This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

This project uses Taskfile for development commands:

```bash
# Development workflow
task install     # Install Go and dependencies
task lint        # Run golangci-lint (MUST run before commits)
task test        # Run tests with coverage
task test:coverage  # View coverage in browser
task test:reset  # Clear test cache

# Direct Go commands
go test ./...           # Run all tests
go test -v ./... -run TestName  # Run specific test
```

## Architecture

This is Schematic's **Rules Engine** - a feature flag evaluation system compiled to WebAssembly for client-side use. The core evaluation flow is:

1. **flagcheck.go** - Main `CheckFlag()` entry point
2. **rulecheck.go** - Processes individual rules and conditions
3. **models.go** - Core data structures (Flag, Rule, Condition, Company, User)
4. **metrics.go** - Usage tracking and billing cycle management

### Rule Processing Priority

Rules are evaluated in strict priority order:

1. `global_override` - System-wide toggles
2. `company_override` - Company-specific overrides
3. `plan_entitlement` - Plan-based features
4. `company_override_usage_exceeded` - Usage limit overrides
5. `plan_entitlement_usage_exceeded` - Plan usage enforcement
6. `standard` - Regular business rules
7. `default` - Fallback values

### Key Data Flow

**Flag Evaluation:**

- Flags contain multiple Rules ordered by priority
- Rules contain Conditions (AND logic within rule, OR logic within condition groups)
- Conditions check company/user traits, metrics, plans, etc.

**Usage Tracking:**

- CompanyMetric tracks usage over time periods (daily/weekly/monthly/all-time)
- Metric periods can reset on calendar boundaries or billing cycles
- Usage limits trigger rule overrides when exceeded

### Condition Types

- **Company/User targeting** - ID-based matching with set operations
- **Plan checks** - Current plan vs. allowed plans
- **Metric conditions** - Usage comparisons with time periods
- **Trait matching** - Key-value trait comparisons using typeconvert utilities

### WebAssembly Interface

The main.go provides a WASM interface that:

- Exposes `checkFlag` function to JavaScript
- Accepts JSON with company/user/flag data
- Returns evaluation results with detailed error information
- Runs persistently in browser environments

## Key Files

- **flagcheck.go** - Main evaluation logic
- **rulecheck.go** - Rule/condition processing
- **metrics.go** - Time-based usage tracking and billing cycles
- **models.go** - All data structures with JSON serialization
- **typeconvert/** - Type comparison utilities for conditions
- **set/** - Generic set operations for ID matching
