# UI theming and white-labeling

The UI uses IBM Carbon semantic tokens as the design-system source of truth and
Tailwind CSS as a utility-class interface over those tokens.

## Runtime configuration

The application loads `/theme.json` before React renders. The request is not
cached and is aborted after three seconds. Missing, invalid, or unavailable
configuration falls back to the built-in theme.

Deployments can replace `public/theme.json` without rebuilding the JavaScript
bundle. Keep the file on the same origin as the UI.

The provided UI container also supports mounting the manifest at
`/app/theme.json`; the entrypoint installs it into the static build before the
server starts:

```bash
docker run --mount type=bind,src=$PWD/theme.json,dst=/app/theme.json,readonly ...
```

For subpath deployments, set the application's public URL during the build. The
runtime manifest URL follows that public base path.

```json
{
  "schemaVersion": 1,
  "id": "customer-a",
  "brand": {
    "name": "Customer A",
    "logos": {
      "full": {
        "light": "/branding/logo-dark-text.svg",
        "dark": "/branding/logo-light-text.svg"
      },
      "compact": {
        "light": "/branding/mark.svg",
        "dark": "/branding/mark.svg"
      }
    },
    "favicon": "/branding/favicon.ico"
  },
  "links": {
    "documentation": "https://docs.example.com",
    "source": "https://github.com/example/project",
    "support": "mailto:support@example.com",
    "terms": "/legal/terms",
    "privacy": "/legal/privacy"
  },
  "defaultMode": "system",
  "allowModeSelection": true,
  "colors": {
    "light": {
      "primary": "#0f62fe",
      "primaryHover": "#0043ce",
      "primaryActive": "#002d9c",
      "onPrimary": "#ffffff"
    },
    "dark": {
      "primary": "#78a9ff",
      "primaryHover": "#a6c8ff",
      "primaryActive": "#d0e2ff",
      "onPrimary": "#161616"
    }
  }
}
```

## Asset requirements

- Full logos should be horizontal SVG or PNG assets with transparent backgrounds.
- Compact logos should be square and remain legible at 24 by 24 pixels.
- Asset URLs must be same-origin paths or HTTP(S) URLs.
- Product links accept same-origin paths, HTTP(S), and `mailto:` URLs.
- Brand colors use six-digit hexadecimal values.
- `onPrimary` must maintain WCAG AA contrast against primary, hover, and active colors.

## Theme behavior

- `defaultMode` accepts `light`, `dark`, or `system`.
- `allowModeSelection: false` locks the deployment to its configured mode.
- User preference is stored under `ui-theme-mode`.
- The resolved mode is exposed through `data-color-mode` on `<html>`.
- The active tenant identifier is exposed through `data-brand` on `<html>`.

## Styling rules

Use Carbon components for controls and Carbon semantic roles for application
surfaces. Tailwind classes should describe intent rather than palette values.

Preferred classes include:

- `bg-shell` for navigation chrome
- `bg-surface` for page backgrounds
- `bg-layer` and `bg-layer-hover` for elevated content
- `text-foreground` and `text-muted`
- `border-border-subtle` and `border-border-strong`
- `text-primary`, `bg-primary`, and related interaction utilities

Avoid adding structural combinations such as `bg-white dark:bg-gray-900` or
`border-gray-200 dark:border-gray-800`. Status colors may use explicit semantic
red, green, yellow, or blue roles when they communicate state rather than
application structure.

## Validation

Run the theme contract check, tests, and production build before submitting UI
changes:

```bash
yarn check:theme
yarn test --watchAll=false --runInBand
yarn build
```

Schema changes require incrementing `schemaVersion`, updating the runtime
validator and default manifest together, and documenting the migration here.
