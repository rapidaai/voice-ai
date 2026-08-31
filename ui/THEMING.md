# UI theming and white-labeling

The UI uses IBM Carbon semantic tokens as the design-system source of truth and
Tailwind CSS as a utility-class interface over those tokens.

## Build configuration

Branding is part of the existing UI configuration contract. Add the complete
theme object under the root `theme` key in the config used for the build:

- `src/configs/config.development.json` for local development
- `src/configs/config.production.json` for the production web build
- `docker/ui/config.<edition>.json` for Docker edition builds

The selected configuration is bundled with the client. White-label deployments
therefore update the selected config and rebuild the UI image or static bundle.
There is no separate runtime theme request, cache, or HTML bootstrap script.

The static Web App Manifest was removed because it duplicated brand names,
icons, and colors outside this contract. If installable-app metadata is needed
later, generate it from the selected build config rather than adding another
hand-maintained branding source.

The optional `/app/config.ui.json` mount continues to replace only the domain
placeholder at container startup. It does not change branding or theme values.

```json
{
  "theme": {
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
}
```

## Asset requirements

- Full logos should be horizontal SVG or PNG assets with transparent backgrounds.
- Compact logos should be square and remain legible at 24 by 24 pixels.
- Asset and product-link fields preserve arbitrary non-empty string values so
  deployments can use tenant-specific schemes, tokens, or routing conventions.
- Brand color fields preserve arbitrary non-empty string values, including CSS
  color functions, custom properties, and tenant-specific tokens.
- Theme loading does not validate color format or contrast. Deployments own
  browser compatibility and accessible contrast for their configured values.

## Theme behavior

- `defaultMode` accepts `light`, `dark`, or `system`.
- `allowModeSelection: false` ignores stored and cross-tab user overrides.
- A locked `system` mode continues to follow operating-system mode changes.
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
yarn check:theme-contract
yarn test:theme-contract
yarn test --watchAll=false --runInBand
yarn build
```

The theme contract check also scans renderable UI source for hardcoded Rapida
brand literals. The scan covers `ui/src/app/**/*.ts`, `ui/src/app/**/*.tsx`,
`ui/src/theme/**/*.ts`, `ui/src/theme/**/*.tsx`, and `ui/public/LLMs.txt`, while
excluding tests, provider catalogs, and generated style output. Exceptions are
declared in `scripts/check-theme-contract.mjs` as exact literals with a short
reason. Do not skip an entire file for branding unless every allowed literal is
listed there.

Schema changes require incrementing `schemaVersion`, updating the config
validator and every deployable config together, and documenting the migration
here. Invalid theme configuration fails closed before React renders and shows a
neutral configuration error instead of applying another tenant's branding.
