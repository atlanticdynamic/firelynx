# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "mcp",
# ]
# ///

# client_list_tools.py
import asyncio

from mcp import ClientSession  # type: ignore
from mcp.client.streamable_http import streamablehttp_client as client  # type: ignore

SERVER_URL = "http://localhost:8083/mcp"


async def main() -> None:
    async with client(SERVER_URL) as (read, write, _):
        async with ClientSession(read, write) as session:
            await session.initialize()
            resp = await session.list_tools()
            if len(resp.tools) > 0:
                print_tools(resp.tools)


def print_tools(tools: dict) -> None:
    total = 0
    print("Available MCP tools:")
    print("-" * 79)
    for tool in tools:
        total += 1
        print(f"Name: {tool.name}")
        print(f"Title: {tool.title or '‑'}")
        print(f"Description: {tool.description}")
        print(f"InputSchema: {tool.inputSchema}")
        print("-" * 79)
    print(f"Total tools: {total}")


if __name__ == "__main__":
    asyncio.run(main())

