# Example MCP tool implemented in Starlark

def process():
    # Access runtime args and static config through ctx
    args = ctx.get("args", {})
    operation = args.get("operation", "")
    input_text = args.get("input", "")
    config = ctx.get("data", {}).get("config", {})

    if operation == "":
        return {
            "isError": True,
            "content": "Operation is required"
        }
    else:
        if operation == "echo":
            return {
                "isError": False,
                "content": input_text
            }
        elif operation == "count_chars":
            return {
                "isError": False,
                "content": len(input_text)
            }
        elif operation == "split_words":
            return {
                "isError": False,
                "content": input_text.split()
            }
        else:
            return {
                "isError": True,
                "content": "Unsupported operation: " + operation
            }

# Execute the function and assign result to '_'
result = process()
_ = result