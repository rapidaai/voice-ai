# Build stage
FROM node:22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache curl

# Copy package files and install dependencies
COPY ui/package.json ui/yarn.lock ./
RUN yarn install --frozen-lockfile --cache-folder .yarn-cache && \
    rm -rf .yarn-cache

# Copy all necessary source files
COPY ui/.babel-plugin-macrosrc.js ./
COPY ui/.dockerignore ./
COPY ui/.editorconfig ./
COPY ui/.env.production ./
COPY ui/.eslintrc.js ./
COPY ui/.gitattributes ./
COPY ui/.gitignore ./
COPY ui/.npmrc ./
COPY ui/.nvmrc ./
COPY ui/.pre-commit-config.yaml ./
COPY ui/.prettierignore ./
COPY ui/.prettierrc ./
COPY ui/.stylelintrc ./
COPY ui/commitlint.config.js ./
COPY ui/craco.config.js ./
COPY ui/package.json ./
COPY ui/postcss.config.js ./
COPY ui/tsconfig.json ./
COPY ui/yarn.lock ./

# Copy directories for public and source files
COPY ui/public ./public
COPY ui/src ./src

# Expose the port to serve the app
EXPOSE 3000

# Run React development server
CMD ["yarn", "start"]