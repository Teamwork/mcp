#! /usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Tool to interact with different LLM models and the Teamwork.com MCP server.
"""

import argparse
import asyncio
import os

from langchain_mcp_adapters.client import MultiServerMCPClient
from langchain_mcp_adapters.tools import load_mcp_tools
from langgraph.prebuilt import create_react_agent

async def main():
  """Main function to run the example."""

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
    agent = create_react_agent(args.llm_model, tools)

    while True:
      user_input = input("tw-client> ")
      if user_input.lower() == 'exit':
        print("Chat ended. Goodbye!")
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