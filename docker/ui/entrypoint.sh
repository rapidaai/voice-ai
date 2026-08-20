#!/bin/sh
set -e

BUILD_DIR="/app/build"
CONFIG_FILE="/app/config.ui.json"
THEME_FILE="/app/theme.json"

# Replace the default runtime theme when a deployment-specific manifest is
# mounted at /app/theme.json. The temporary file avoids serving partial writes.
if [ -f "$THEME_FILE" ]; then
  echo "Installing runtime theme manifest..."
  cp "$THEME_FILE" "$BUILD_DIR/theme.json.tmp"
  mv "$BUILD_DIR/theme.json.tmp" "$BUILD_DIR/theme.json"
fi

# If a runtime config is mounted, inject it into the built JS bundle
# by replacing the placeholder values that were baked at build time
if [ -f "$CONFIG_FILE" ]; then
  echo "Injecting runtime config into production build..."
  # Replace DOMAIN_PLACEHOLDER in all JS files with actual domain from mounted config
  DOMAIN=$(grep -o '"domain" *: *"[^"]*"' "$CONFIG_FILE" | head -1 | sed 's/.*: *"\([^"]*\)"/\1/')
  if [ -n "$DOMAIN" ] && [ "$DOMAIN" != "DOMAIN_PLACEHOLDER" ]; then
    find "$BUILD_DIR/static/js" -name '*.js' -exec sed -i "s|DOMAIN_PLACEHOLDER|$DOMAIN|g" {} +
    echo "Config injected: domain=$DOMAIN"
  fi
fi

echo "Serving UI on port 3000..."
exec serve -s build -l 3000
