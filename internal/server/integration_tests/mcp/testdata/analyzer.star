def analyze():
    # Get MCP arguments
    args = ctx.get("args", {})
    data = args.get("data", {})
    analysis_type = args.get("type", "structure")
    
    if not data:
        return {"error": "No data provided"}
    
    if analysis_type == "structure":
        if type(data) == "dict":
            keys = list(data.keys())
            return {
                "text": "Object with {} keys".format(len(keys)),
                "type": "object",
                "key_count": len(keys),
                "keys": keys[:5]  # First 5 keys
            }
        elif type(data) == "list":
            return {
                "text": "Array with {} items".format(len(data)),
                "type": "array",
                "length": len(data)
            }
        else:
            return {
                "text": "Primitive value: {}".format(str(data)),
                "type": type(data)
            }
    else:
        return {"error": "Unknown analysis type"}

result = analyze()
_ = result