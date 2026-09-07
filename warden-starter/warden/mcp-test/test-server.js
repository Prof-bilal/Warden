#!/usr/bin/env node
/**
 * Simple MCP server for testing Warden sandbox
 * This server performs basic filesystem operations to test sandboxing
 */

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');
const { 
  CallToolRequestSchema, 
  ListToolsRequestSchema,
} = require('@modelcontextprotocol/sdk/types.js');

const fs = require('fs');
const path = require('path');

// Simple filesystem MCP server
const server = new Server(
  {
    name: "warden-test-server",
    version: "1.0.0",
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: "read_file",
        description: "Read contents of a file",
        inputSchema: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "Path to the file to read"
            }
          },
          required: ["path"]
        }
      },
      {
        name: "write_file",
        description: "Write contents to a file",
        inputSchema: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "Path to the file to write"
            },
            content: {
              type: "string",
              description: "Content to write"
            }
          },
          required: ["path", "content"]
        }
      },
      {
        name: "list_directory",
        description: "List files in a directory",
        inputSchema: {
          type: "object",
          properties: {
            path: {
              type: "string",
              description: "Path to the directory"
            }
          },
          required: ["path"]
        }
      },
      {
        name: "test_network",
        description: "Test network connectivity",
        inputSchema: {
          type: "object",
          properties: {
            host: {
              type: "string",
              description: "Host to test"
            }
          },
          required: ["host"]
        }
      }
    ]
  };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;
  
  try {
    switch (name) {
      case "read_file": {
        const content = fs.readFileSync(args.path, 'utf-8');
        return {
          content: [
            {
              type: "text",
              text: `File contents of ${args.path}:\n${content}`
            }
          ]
        };
      }
      
      case "write_file": {
        fs.writeFileSync(args.path, args.content);
        return {
          content: [
            {
              type: "text",
              text: `Successfully wrote to ${args.path}`
            }
          ]
        };
      }
      
      case "list_directory": {
        const files = fs.readdirSync(args.path);
        return {
          content: [
            {
              type: "text",
              text: `Files in ${args.path}:\n${files.join('\n')}`
            }
          ]
        };
      }
      
      case "test_network": {
        // This should be blocked by the sandbox if host is not allowlisted
        const net = require('net');
        return {
          content: [
            {
              type: "text",
              text: `Attempted network test to ${args.host}`
            }
          ]
        };
      }
      
      default:
        return {
          content: [
            {
              type: "text",
              text: `Unknown tool: ${name}`
            }
          ],
          isError: true
        };
    }
  } catch (error) {
    return {
      content: [
        {
          type: "text",
          text: `Error: ${error.message}`
        }
      ],
      isError: true
    };
  }
});

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error("Warden test MCP server started");
}

main().catch(console.error);
