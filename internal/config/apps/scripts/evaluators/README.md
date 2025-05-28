# Script Evaluators

The `evaluators` package implements script engine configurations for different scripting languages.

## Evaluator Types

- **RisorEvaluator**: Configuration for Risor scripts
- **StarlarkEvaluator**: Configuration for Starlark scripts  
- **ExtismEvaluator**: Configuration for WebAssembly modules

## Validation

Each evaluator validates:
- Script syntax correctness
- Required fields presence
- Engine-specific constraints

## Integration

Evaluators are used by script apps to define which engine processes the script code. The actual script execution happens in the server layer using go-polyscript.