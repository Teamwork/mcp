#! /usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Tool to interact with different LLM models and the Teamwork.com MCP server.
"""

import argparse
import asyncio
import os
import signal
import sys

from langchain.agents import create_agent
from langchain_mcp_adapters.client import MultiServerMCPClient
from langchain_mcp_adapters.tools import load_mcp_tools

def signal_handler(signal, frame):
    """Handle termination signals to gracefully exit."""
    print("👋 Goodbye!")
    sys.exit(0)

async def main():
  """Main function to run the example."""

  signal.signal(signal.SIGINT, signal_handler)
  signal.signal(signal.SIGTERM, signal_handler)

  parser = argparse.ArgumentParser(description="Run the Teamwork.com MCP client example.")
  parser.add_argument(
    "--server",
    type=str,
    default="https://mcp.ai.teamwork.com",
    help="The MCP server URL to connect to (default: https://mcp.ai.teamwork.com)",
  )
  parser.add_argument(
    "--bearer-token",
    type=str,
    default=os.getenv("TW_MCP_BEARER_TOKEN", ""),
    help="Bearer token for authentication with the MCP server (default: from environment variable TW_MCP_BEARER_TOKEN)",
  )
  parser.add_argument(
    "--llm-model",
    type=str,
    default="openai:gpt-4.1",
    help="The LLM model to use (default: openai:gpt-4.1)",
  )
  args = parser.parse_args()

  if not args.server:
    print("Error: Please provide a server URL using the --server argument.")
    return
  if not args.bearer_token:
    print("Error: Please provide a bearer token using the --bearer-token argument or set the TW_MCP_BEARER_TOKEN environment variable.")
    return
  if not args.llm_model:
    print("Error: Please provide an LLM model using the --llm-model argument.")
    return

  client = MultiServerMCPClient(
    {
      "Teamwork.com": {
        "transport": "streamable_http",
        "url": args.server,
        "headers": {
          "Authorization": "Bearer " + args.bearer_token,
        }
      },
    }
  )

  async with client.session("Teamwork.com") as session:
    tools = await load_mcp_tools(session)
    agent = create_agent(args.llm_model, tools)

    while True:
      try:
        user_input = input("tw-client> ")
      except EOFError:
        print("\n👋 Goodbye!")
        break

      if user_input.lower() == 'exit':
        print("👋 Goodbye!")
        break

      response = await agent.ainvoke({"messages": user_input})

      messages = response.get("messages", [])
      if messages:
        for message in reversed(messages):
          if hasattr(message, 'content') and message.__class__.__name__ == 'AIMessage':
            print(message.content)
            break
      else:
        print("No response received.")

if __name__ == "__main__":
  asyncio.run(main())