import { Command } from 'commander';
import { createInterface } from 'readline';

import { ChatAnthropic } from '@langchain/anthropic';
import { ChatGoogleGenerativeAI } from '@langchain/google-genai';
import { ChatOpenAI } from '@langchain/openai';
import { MultiServerMCPClient } from '@langchain/mcp-adapters';
import { createReactAgent } from '@langchain/langgraph/prebuilt';

type ReactAgent = ReturnType<typeof createReactAgent>;

const rl = createInterface({
  input: process.stdin,
  output: process.stdout,
});

function extractFinalAIMessage(response: any): string | null {
  if (!response.messages || !Array.isArray(response.messages)) {
    return null;
  }
  
  const aiMessages = response.messages.filter((msg: any) => msg.constructor.name === 'AIMessage');
  if (aiMessages.length === 0) {
    return null;
  }
  const finalAIMessage = aiMessages[aiMessages.length - 1];
  return finalAIMessage.content;
}

async function askQuestion(agent: ReactAgent, history: any): Promise<void> {
  rl.question('tw-client> ', async (input: string) => {
    if (input.trim().toLowerCase() === 'exit') {
      console.log('👋 Goodbye!');
      rl.close();
      process.exit(0);
    } else if (input.trim() === '') {
      return askQuestion(agent, history);
    }

    const request = [];
    if (history && history.messages) {
      request.push(...history.messages);
    }
    request.push({ role: 'user', content: input });

    try {
      const response = await agent.invoke({
        messages: request,
      });

      const finalMessage = extractFinalAIMessage(response);
      if (!finalMessage) {
        console.log('No response received.');
        rl.close();
        process.exit(0);
      }

      console.log(finalMessage);
      history = response;

    } catch (error) {
      console.error('Error during agent execution:', error);
    }

    return askQuestion(agent, history);
  });
}

async function main() {
  const program = new Command();

  program
    .name('tw-client')
    .description('A CLI tool to interact with Teamwork MCP using LangChain')
    .version('1.0.0')
    .option('-m, --llm-model <model>', 'Model to use', 'openai:gpt-4o-mini')
    .option('-t, --bearer-token <token>', 'API token for the MCP server', process.env.TW_MCP_BEARER_TOKEN || '')
    .option('-s, --server-url <url>', 'URL of the MCP server', 'https://mcp.ai.teamwork.com')
    .parse();

  const options = program.opts();
  if (!options.llmModel) {
    console.error('Error: LLM model is required. Use -m or --llm-model to specify it.');
    process.exit(1);
  }
  if (!options.bearerToken) {
    console.error('Error: Bearer token is required. Use -t or --bearer-token to provide it.');
    process.exit(1);
  }
  if (!options.serverUrl) {
    console.error('Error: Server URL is required. Use -s or --server-url to specify it.');
    process.exit(1);
  }

  const client = new MultiServerMCPClient({
    mcpServers: {
      teamwork: {
        url: options.serverUrl,
        headers: {
          Authorization: `Bearer ${options.bearerToken}`,
        }
      },
    },
  });

  const tools = await client.getTools();

  let model;
  const [provider, modelName] = options.llmModel.split(':');
  if (provider === 'openai') {
    model = new ChatOpenAI({
      model: modelName
    });
  } else if (provider === 'google_genai') {
    model = new ChatGoogleGenerativeAI({
      model: modelName
    });
  } else if (provider === 'anthropic') {
    model = new ChatAnthropic({
      model: modelName
    });
  } else {
    console.error(`Unsupported provider for LLM model: ${provider}. '
      'Supported providers are: openai, google_genai, anthropic.`);
    process.exit(1);
  }

  const agent = createReactAgent({
    llm: model,
    tools,
  });

  await askQuestion(agent, null);
}

main().catch(console.error);