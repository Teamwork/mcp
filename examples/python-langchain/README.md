# Teamwork MCP Python Client Example

This example demonstrates how to interact with the Teamwork MCP server using a
Python client. The client connects to the MCP server and can execute various
Teamwork operations through the Model Context Protocol.

## 📋 Prerequisites

- Python 3.13 or later
- pip (Python package installer)
- Access to a running Teamwork MCP server
- Valid Teamwork API credentials

## ⚙️ Setup

### 1. Create a Virtual Environment

Create and activate a virtual environment:

```bash
cd examples/python-langchain
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

### 2. Install Dependencies

Install the required dependencies:

```bash
pip install -r requirements.txt
```

### 3. Environment Variables

Set up the required environment variables:

```bash
export TW_MCP_BEARER_TOKEN="your-bearer-token-here"
```

Depending on the LLM model you want to use, you may also need to set:

```bash
# when using openai:gpt-4, openai:gpt-4-turbo, openai:gpt-3.5-turbo, etc.
export OPENAI_API_KEY="your-openai-api-key-here"

# when using anthropic:claude-3-5-sonnet-20241022, anthropic:claude-3-opus-20240229, etc.
export ANTHROPIC_API_KEY="your-anthropic-api-key-here"

# when using google_genai:gemini-2.5-flash, etc.
export GOOGLE_API_KEY="your-google-api-key-here"
```

## 🚀 Running the Example

Make sure your virtual environment is activated, then run:

```bash
python main.py --llm-model openai:gpt-4o
```

Other options are:
* `--server`: The MCP server URL to connect to (default:
  `https://mcp.ai.teamwork.com`)

* `--bearer-token`: Bearer token for authentication with the MCP server
  (default: from environment variable `TW_MCP_BEARER_TOKEN`)

* `--llm-model`: The LLM model to use (default: `openai:gpt-4.1`)

## 🤓 Output example

```shell
tw-client> Could you list all tasks?
Here are the tasks currently listed:

1. **Task Name:** Create Marketing Plan
   - **Description:** Develop a marketing strategy for the game release.
   - **Status:** New

2. **Task Name:** Create Game Concept
   - **Description:** Outline the main idea, theme, and gameplay mechanics.
   - **Status:** New

3. **Task Name:** Develop Game Prototype
   - **Description:** Create a basic version of the game for testing concepts.
   - **Status:** New

4. **Task Name:** Design Game Characters
   - **Description:** Design main characters and their backstories.
   - **Status:** New

5. **Task Name:** Conduct Alpha Testing
   - **Description:** Test game features and gather feedback from a closed group.
   - **Status:** New

If you need more details or further assistance, feel free to ask!

tw-client> Give me more details of the second task
The second task is titled "Create Game Concept." Here's more information about this task:

- **Description**: Outline the main idea, theme, and gameplay mechanics.
- **Priority**: Not specified
- **Progress**: 0% complete
- **Start Date**: Not specified
- **Due Date**: Not specified
- **Estimated Time (Minutes)**: 0 minutes
- **Tasklist ID**: 1415150
- **Status**: New
- **Assignees**: None assigned
- **Tags**: None

If you need more details or assistance with this task, feel free to ask!

tw-client> exit
Chat ended. Goodbye!
```