# ChatGPT Apps SDK Wrapper for ToolBridge
# Thin Node.js layer that adds Apps SDK metadata to Python MCP backend

FROM node:20-slim AS builder

WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./

# Install dependencies
RUN npm ci

# Copy source
COPY tsconfig.json ./
COPY src ./src

# Build TypeScript
RUN npm run build

# ════════════════════════════════════════════════════════════════════════════
# Production image
# ════════════════════════════════════════════════════════════════════════════

FROM node:20-slim

WORKDIR /app

# Copy package files and install production deps only
COPY package.json package-lock.json* ./
RUN npm ci --omit=dev

# Copy built files from builder
COPY --from=builder /app/dist ./dist

# Create non-root user
RUN useradd -m -u 1000 app && chown -R app:app /app
USER app

# Environment variables
ENV NODE_ENV=production
ENV PYTHON_MCP_URL=https://toolbridge-mcp-staging.fly.dev/mcp

# Start server
CMD ["node", "dist/server.js"]
