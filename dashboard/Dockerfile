# Build stage using bun to match the developer's package manager
FROM oven/bun:1-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY package.json bun.lock ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy source code and config files
COPY . .

# Build the Static Vite React app
RUN bun run build

# Runtime Stage using Nginx
FROM nginx:alpine

# Remove default nginx config
RUN rm /etc/nginx/conf.d/default.conf

# Copy custom Nginx configuration that proxies /api and /ws to the Go backend
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy the built production assets from the bun builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Run nginx in foreground
CMD ["nginx", "-g", "daemon off;"]
