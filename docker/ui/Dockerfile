# Build stage
FROM node:22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache curl

# Copy package files and install deps
COPY ui/package.json ui/yarn.lock ./

# Copy only source code
COPY ui/.babel-plugin-macrosrc.js ./
COPY ui/.dockerignore ./
COPY ui/.editorconfig ./
COPY ui/.env.production ./
COPY ui/.eslintignore ./
COPY ui/.eslintrc.js ./
COPY ui/.gitattributes ./
COPY ui/.gitignore ./
COPY ui/.npmrc ./
COPY ui/.nvmrc ./
COPY ui/.pre-commit-config.yaml ./
COPY ui/.prettierignore ./
COPY ui/.prettierrc ./
COPY ui/.stylelintrc ./
COPY ui/amplify.yml ./
COPY ui/commitlint.config.js ./
COPY ui/craco.config.js ./
COPY ui/electron-builder.yml ./
COPY ui/package.json ./
COPY ui/postcss.config.js ./
COPY ui/template.json ./
COPY ui/tsconfig.json ./
COPY ui/tsconfig.paths.json ./
COPY ui/yarn.lock ./

# RUN yarn install --frozen-lockfile
RUN yarn install --frozen-lockfile --cache-folder .yarn-cache && \
    rm -rf .yarn-cache

# Copy directories
COPY ui/public/ ./public
COPY ui/src/ ./src


# Build the React app
RUN yarn build:docker

# Final stage - Alpine (just holds the build)
FROM alpine:latest

# Copy built files from builder
COPY --from=builder /app/build /build

# Keep container running
CMD cp -r /build/* /usr/share/nginx/html/ 2>/dev/null; tail -f /dev/null