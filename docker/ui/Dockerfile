# Build stage
FROM node:22-alpine AS builder

WORKDIR /app

# Copy package files
COPY ui/package.json ui/yarn.lock ./

# Install dependencies
RUN yarn install

# Copy source code
COPY ui/ .

# Build the React app
RUN export NODE_ENV="production" && \
    export PATH="$PATH:./node_modules/.bin" && \
    yarn run build

# Final stage - Nginx
FROM nginx:alpine

# Create rapida-app user
RUN addgroup -g 1000 rapida-app && \
    adduser -D -u 1000 -G rapida-app rapida-app

# Copy nginx config
COPY docker/nginx/nginx.conf /etc/nginx/conf.d/default.conf

# Copy built app from builder
COPY --from=builder /app/build /usr/share/nginx/html

# Copy images directory if it exists
COPY --from=builder /app/public/images /usr/share/nginx/html/images

# Create directory for assets
RUN mkdir -p /usr/share/nginx/html/assets && \
    chown -R rapida-app:rapida-app /usr/share/nginx/html && \
    chown -R rapida-app:rapida-app /var/cache/nginx && \
    chown -R rapida-app:rapida-app /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R rapida-app:rapida-app /var/run/nginx.pid

# Switch to rapida-app user
USER rapida-app

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

# Run nginx
CMD ["nginx", "-g", "daemon off;"]