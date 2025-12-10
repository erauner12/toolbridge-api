/**
 * MCP Client to proxy requests to the Python ToolBridge MCP server.
 *
 * Uses the MCP SDK's StreamableHTTPClientTransport for proper session handling.
 * The Python backend (FastMCP) uses stateful sessions, so we need to establish
 * a connection and maintain the session ID.
 */

import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";

interface ToolCallResult {
  content: Array<{
    type: string;
    text?: string;
    resource?: {
      uri: string;
      mimeType: string;
      text: string;
    };
  }>;
  structuredContent?: unknown;
  isError?: boolean;
}

export class McpClient {
  private baseUrl: string;
  private client: Client | null = null;
  private transport: StreamableHTTPClientTransport | null = null;
  private accessToken?: string;
  private connectionPromise: Promise<void> | null = null;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  /**
   * Set the OAuth access token for authenticated requests.
   * This should be called when the user authenticates.
   * Note: Changing the token requires reconnecting.
   */
  setAccessToken(token: string): void {
    // If token changed, we need to reconnect
    if (this.accessToken !== token) {
      this.accessToken = token;
      // Force reconnection with new token
      this.disconnect();
    }
  }

  /**
   * Disconnect from the Python backend.
   */
  private disconnect(): void {
    if (this.transport) {
      this.transport.close().catch(() => {});
      this.transport = null;
    }
    this.client = null;
    this.connectionPromise = null;
  }

  /**
   * Ensure we have an active connection to the Python backend.
   */
  private async ensureConnected(): Promise<Client> {
    // If already connecting, wait for that
    if (this.connectionPromise) {
      await this.connectionPromise;
      if (this.client) {
        return this.client;
      }
    }

    // If already connected, return the client
    if (this.client) {
      return this.client;
    }

    // Start new connection
    this.connectionPromise = this.connect();
    await this.connectionPromise;

    if (!this.client) {
      throw new Error("Failed to connect to MCP backend");
    }

    return this.client;
  }

  /**
   * Connect to the Python MCP backend using StreamableHTTPClientTransport.
   */
  private async connect(): Promise<void> {
    console.error(`[McpClient] Connecting to ${this.baseUrl}`);

    const url = new URL(this.baseUrl);

    // Create transport with auth token in request headers
    const requestInit: RequestInit = {
      headers: {
        ...(this.accessToken
          ? { Authorization: `Bearer ${this.accessToken}` }
          : {}),
      },
    };

    this.transport = new StreamableHTTPClientTransport(url, {
      requestInit,
    });

    // Create MCP client
    this.client = new Client(
      {
        name: "toolbridge-apps-wrapper",
        version: "1.0.0",
      },
      {
        capabilities: {},
      }
    );

    // Connect client to transport
    await this.client.connect(this.transport);

    console.error(
      `[McpClient] Connected to backend, session: ${this.transport.sessionId}`
    );
  }

  /**
   * Call a tool on the Python MCP backend.
   */
  async callTool(
    name: string,
    args: Record<string, unknown>
  ): Promise<ToolCallResult> {
    console.error(`[McpClient] tools/call ${name}`, JSON.stringify(args));

    const client = await this.ensureConnected();

    const result = await client.callTool({
      name,
      arguments: args,
    });

    console.error(`[McpClient] Tool result received`);

    return result as ToolCallResult;
  }

  /**
   * Read a resource (HTML widget) from the Python MCP backend.
   * Returns the HTML content as a string.
   */
  async readResource(uri: string): Promise<string> {
    console.error(`[McpClient] resources/read ${uri}`);

    const client = await this.ensureConnected();

    const result = await client.readResource({ uri });

    if (!result.contents || result.contents.length === 0) {
      throw new Error(`No content returned for resource: ${uri}`);
    }

    const content = result.contents[0];
    if ("text" in content) {
      return content.text;
    }

    throw new Error(`Resource content is not text: ${uri}`);
  }

  /**
   * List available tools from the Python MCP backend.
   */
  async listTools(): Promise<unknown[]> {
    const client = await this.ensureConnected();
    const result = await client.listTools();
    return result.tools;
  }

  /**
   * List available resources from the Python MCP backend.
   */
  async listResources(): Promise<unknown[]> {
    const client = await this.ensureConnected();
    const result = await client.listResources();
    return result.resources;
  }
}
