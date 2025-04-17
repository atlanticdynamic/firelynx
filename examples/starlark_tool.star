# Example MCP tool implemented in Starlark

# Access parameters and static config through ctx
operation = ctx.get("operation", "")
input_text = ctx.get("input", "")
config = ctx.get("config", {})

if operation == "":
    result = {
        "isError": True,
        "content": "Operation is required"
    }
else:
    if operation == "echo":
        result = {
            "isError": False,
            "content": input_text
        }
    elif operation == "count_chars":
        result = {
            "isError": False,
            "content": len(input_text)
        }
    elif operation == "split_words":
        result = {
            "isError": False,
            "content": input_text.split()
        }
    else:
        result = {
            "isError": True,
            "content": "Unsupported operation: " + operation
        }

# Return the result to Starlark by assigning to '_'
_ = result
