/**
 * MCP Client to proxy requests to the Python ToolBridge MCP server.
 *
 * This client handles:
 * - Tool calls (tools/call)
 * - Resource reads (resources/read)
 * - Authentication token forwarding
 */

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

interface McpRequest {
  jsonrpc: "2.0";
  id: number;
  method: string;
  params?: Record<string, unknown>;
}

interface McpResponse {
  jsonrpc: "2.0";
  id: number;
  result?: unknown;
  error?: {
    code: number;
    message: string;
  };
}

export class McpClient {
  private baseUrl: string;
  private requestId = 0;
  private accessToken?: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  /**
   * Set the OAuth access token for authenticated requests.
   * This should be called when the user authenticates.
   */
  setAccessToken(token: string): void {
    this.accessToken = token;
  }

  /**
   * Call a tool on the Python MCP backend.
   */
  async callTool(name: string, args: Record<string, unknown>): Promise<ToolCallResult> {
    const response = await this.sendRequest("tools/call", {
      name,
      arguments: args,
    });

    return response as ToolCallResult;
  }

  /**
   * Read a resource (HTML widget) from the Python MCP backend.
   * Returns the HTML content as a string.
   */
  async readResource(uri: string): Promise<string> {
    const response = await this.sendRequest("resources/read", { uri });

    // Extract HTML from response
    const result = response as {
      contents: Array<{
        uri: string;
        mimeType: string;
        text: string;
      }>;
    };

    if (!result.contents || result.contents.length === 0) {
      throw new Error(`No content returned for resource: ${uri}`);
    }

    return result.contents[0].text;
  }

  /**
   * List available tools from the Python MCP backend.
   */
  async listTools(): Promise<unknown[]> {
    const response = await this.sendRequest("tools/list", {});
    return (response as { tools: unknown[] }).tools;
  }

  /**
   * List available resources from the Python MCP backend.
   */
  async listResources(): Promise<unknown[]> {
    const response = await this.sendRequest("resources/list", {});
    return (response as { resources: unknown[] }).resources;
  }

  /**
   * Send a JSON-RPC request to the Python MCP backend.
   */
  private async sendRequest(method: string, params: Record<string, unknown>): Promise<unknown> {
    const request: McpRequest = {
      jsonrpc: "2.0",
      id: ++this.requestId,
      method,
      params,
    };

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    // Add auth token if available
    if (this.accessToken) {
      headers["Authorization"] = `Bearer ${this.accessToken}`;
    }

    console.log(`[McpClient] ${method}`, params);

    const response = await fetch(this.baseUrl, {
      method: "POST",
      headers,
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`MCP request failed: ${response.status} ${response.statusText}`);
    }

    const data = (await response.json()) as McpResponse;

    if (data.error) {
      throw new Error(`MCP error: ${data.error.message} (code: ${data.error.code})`);
    }

    return data.result;
  }
}
