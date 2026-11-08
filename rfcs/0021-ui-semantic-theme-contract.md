# RFC 0021: UI Semantic Theme Contract

- Status: Accepted
- Owner: UI Platform
- Approved by: Repository owner
- Date: 2026-11-06
- Supersedes: RFC 0014 (color contract only)

## Context

The UI currently has three color owners. `CONFIG.theme` owns brand identity and four
primary-action values, Carbon supplies most semantic component colors, and renderable
components contain Tailwind palette utilities and light/dark color pairs. The same role
can therefore render differently in Carbon controls, shared primitives, and page
components.

RFC 0014 made `CONFIG.theme` authoritative for UI branding, but it intentionally kept
the existing version 1 color contract. That contract does not select the Carbon base
theme and does not guarantee that Carbon components and Tailwind utilities resolve a
role from the same value.

The UI uses IBM Carbon as its design system and Tailwind CSS as its utility-class
interface. Component authors should compose Carbon components and Tailwind classes
without writing component CSS. The public configuration must remain smaller and more
stable than Carbon's internal token inventory.

RFC 0014 remains the accepted owner of brand names, logos, favicons, product links,
fallback assets, and public metadata. This RFC replaces only its color-contract decision.
RFC 0014 must not be marked fully superseded when this RFC is accepted.

### Non-goals

- Backend email branding, API behavior, authentication, authorization, or stored data.
- Runtime tenant switching without rebuilding the selected UI bundle.
- Provider names, provider logos, repository identity, package names, or legal identity.
- Public configuration of individual Carbon custom properties.
- Product workflow, information-architecture, or feature redesign.

## Decision

Introduce `schemaVersion: 2` for the UI theme manifest. The selected deployable
`CONFIG.theme` is the source of truth for a built deployment. Each mode selects a Carbon
base theme and supplies the brand interaction ramp. The resolved Carbon semantic custom
properties are the sole runtime color source consumed by Carbon components, Tailwind
utilities, shared primitives, charts, editors, and page components.

The version 2 color contract is exactly:

```ts
interface ThemeColorsV2 {
  light: {
    carbonTheme: "white" | "g10";
    primary: string;
    primaryHover: string;
    primaryActive: string;
    onPrimary: string;
  };
  dark: {
    carbonTheme: "g90" | "g100";
    primary: string;
    primaryHover: string;
    primaryActive: string;
    onPrimary: string;
  };
}
```

All other version 1 identity, asset, link, default-mode, and mode-selection fields remain
part of version 2 without semantic changes. Version 2 adds no arbitrary semantic-token
map and exposes no raw `--cds-*` configuration keys.

The following table is the normative version 2 color schema. Every row is required.

| Role                     | JSON path                          | CSS output or document output         | Version 1 derivation                                                                  |
| ------------------------ | ---------------------------------- | ------------------------------------- | ------------------------------------------------------------------------------------- |
| Light Carbon preset      | `theme.colors.light.carbonTheme`   | `data-carbon-theme`; `white` or `g10` | Fixed to `white`                                                                      |
| Light primary            | `theme.colors.light.primary`       | `--brand-primary`                     | Configured value, otherwise `#525252`                                                 |
| Light primary hover      | `theme.colors.light.primaryHover`  | `--brand-primary-hover`               | Configured value, otherwise `#393939`                                                 |
| Light primary active     | `theme.colors.light.primaryActive` | `--brand-primary-active`              | Configured value, otherwise `#262626`                                                 |
| Light primary foreground | `theme.colors.light.onPrimary`     | `--brand-on-primary`                  | Configured value, otherwise `#ffffff`                                                 |
| Dark Carbon preset       | `theme.colors.dark.carbonTheme`    | `data-carbon-theme`; `g90` or `g100`  | Fixed to `g100`                                                                       |
| Dark primary             | `theme.colors.dark.primary`        | `--brand-primary`                     | Configured value, otherwise `#c6c6c6`; version 1 primary buttons retain the g100 ramp |
| Dark primary hover       | `theme.colors.dark.primaryHover`   | `--brand-primary-hover`               | Configured value, otherwise `#e0e0e0`; version 1 primary buttons retain the g100 ramp |
| Dark primary active      | `theme.colors.dark.primaryActive`  | `--brand-primary-active`              | Configured value, otherwise `#f4f4f4`; version 1 primary buttons retain the g100 ramp |
| Dark primary foreground  | `theme.colors.dark.onPrimary`      | `--brand-on-primary`                  | Configured value, otherwise `#161616`; version 1 primary buttons retain white content |

`ui/src/theme/theme-contract.json` is the machine-readable owner of the schema version,
required color-field names, allowed light and dark Carbon presets, document attributes,
storage key, and brand property names. The theme parser and Node ESM contract checker both
consume that file. TypeScript types derive their keys from its object keys, and parity
tests reject drift between the JSON contract, runtime parser, deployable configs, and CSS
adapters.

The active Carbon preset owns backgrounds, layers, fields, content, borders, secondary
and danger actions, support states, selected states, disabled states, and inverse roles.
The brand ramp intentionally overrides these Carbon roles in version 2:

| Brand value     | Exact Carbon custom properties and scope                                                                                                                            |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `primary`       | `--cds-background-brand`, `--cds-interactive`, `--cds-button-primary`, `--cds-button-tertiary`, `--cds-focus`, `--cds-border-interactive`, and `--cds-link-primary` |
| `primaryHover`  | `--cds-button-primary-hover`, `--cds-button-tertiary-hover`, and `--cds-link-primary-hover`                                                                         |
| `primaryActive` | `--cds-button-primary-active` and `--cds-button-tertiary-active`                                                                                                    |
| `onPrimary`     | Selector-scoped `--cds-text-on-color` and `--cds-icon-on-color` on primary buttons and on hovered or active tertiary brand surfaces only                            |

Carbon remains the owner for every role not listed in the override table. This is one
resolved runtime source rather than a second application palette: Tailwind semantic
utilities alias active Carbon properties or the selector-scoped brand properties from the
same mapping and never carry independent color values. `onPrimary` must not override the
root `--cds-text-on-color` or `--cds-icon-on-color` properties. The Carbon adapter scopes
it to primary controls and tertiary controls while their brand background is present, so
danger and support foregrounds remain owned by the selected Carbon preset.

The existing `--brand-primary`, `--brand-primary-hover`, `--brand-primary-active`, and
`--brand-on-primary` names remain the only application-owned color properties. The active
Carbon preset owns the `--cds-*` namespace. No `--app-*` color namespace is introduced.
Version 2 rejects values that reference `var(--cds-*)` or `var(--brand-*)` to prevent
public coupling and cycles. It accepts `color-mix()` and tenant-owned custom properties
such as `var(--tenant-action)` when they do not reference those internal namespaces.
Version 1 retains its existing opaque-value behavior for compatibility.

The styling path is one-way:

1. The selected `CONFIG.theme` supplies identity, assets, links, mode policy, Carbon base
   themes, and brand interaction values.
2. The theme parser validates the selected schema and returns either a complete resolved
   theme or structured issues.
3. The theme provider selects the active Carbon preset and applies the version-specific
   brand mapping before React renders.
4. Tailwind semantic aliases read the resulting Carbon properties.
5. Components use Carbon controls and Tailwind semantic classes only.

Carbon React components are required for controls, form fields, navigation, tables,
notifications, overlays, loading states, and other patterns that Carbon supplies. Domain
composites may combine Carbon primitives, but must preserve Carbon states, keyboard
behavior, type roles, spacing, and icon conventions. Whether Carbon supplies a suitable
pattern is an independent review decision, not a static-checker claim.

Tailwind is the component-authoring interface for layout, responsive behavior, spacing,
and semantic roles. Renderable components must not add component stylesheets, inline
color values, arbitrary color utilities, direct `--cds-*` color expressions, or
palette-specific structural classes. Structural `dark:` color pairs are prohibited
because mode differences belong to the resolved Carbon theme.

Handwritten styling is restricted to central ownership boundaries:

- `ui/src/styles/tailwind.css` declares Tailwind aliases and global utilities;
- `ui/src/styles/carbon.scss` imports Carbon themes and owns the brand override mapping;
- global font and reset files own document-wide foundations;
- verified third-party widgets may use focused adapter rules when no component API or
  Tailwind class can express the integration.

Every third-party exception is an exact path and matched construct in the theme contract
checker, includes a reason, and has focused coverage. Whole-file exclusions are not
allowed.

### Compatibility

Version 1 remains a supported input through one compatibility boundary in
`ui/src/theme/`. It resolves to Carbon white for light mode and g100 for dark mode.

Version 1 preserves existing computed behavior:

- an absent or version 1 schema retains every existing fallback for partial, empty, or
  missing identity, asset, link, mode-policy, and color structures;
- light mode maps the four brand values as it does today;
- dark mode retains the native Carbon g100 primary-button ramp and white primary-button
  text;
- dark brand values continue to control interactive, focus, border, link, and tertiary
  roles as they do today.

Version 2 intentionally changes dark primary buttons to use `primary`, `primaryHover`,
`primaryActive`, and `onPrimary`. Repository-owned configs migrate together with reviewed
light and dark screenshots. External version 1 configs do not receive that visual change.
Strict required-field validation does not apply to version 1. Compatibility tests include
partial, empty, and missing version 1 inputs. Version 1 support can be removed only through
a separately governed change.

Both schema versions preserve `data-color-mode`, `data-theme-ready`, `data-brand`, the
four existing `--brand-*` properties, the `ui-theme-mode` storage key, system-mode
tracking, mode locking, and cross-tab storage synchronization.

### Invalid configuration

`ui/src/theme/theme-config.ts` owns a parser that returns either a resolved theme or a
list of path-specific issues. `ui/src/index.tsx` owns the bootstrap decision. Strict
completeness applies only to version 2. On an unsupported schema, invalid version 2 Carbon
preset, missing version 2 required field, or empty version 2 required value, bootstrap
must:

1. stop before analytics initialization and React root creation;
2. avoid applying brand identifiers, assets, links, colors, or configured text;
3. replace the root contents with fixed neutral text through DOM `textContent` APIs;
4. expose no rejected tenant value in the document or diagnostic shown to the user.

Tests must assert that invalid configuration creates no React root, initializes no
analytics, applies no `data-brand` or brand properties, and renders only the fixed neutral
message. Non-empty CSS values remain opaque so deployments may use CSS functions and
custom properties. Operators own browser support and accessible contrast for custom brand
values. Unsupported but non-empty external values may produce unreadable output; the
runtime does not issue browser-dependent warnings. Browser computed-style and contrast
tests validate every repository-owned config. Invalid JSON is a build-time error from the
Vite JSON loader or Docker image build, so no UI bundle is emitted.

### Static enforcement

The contract checker scans production `.ts` and `.tsx` files under `ui/src/app/` and
`ui/src/theme/`, excluding tests, stories, generated files, and provider catalog data. It
rejects:

- Tailwind color utilities from every built-in literal palette, including status hues;
- arbitrary Tailwind color values and direct Carbon color expressions;
- structural color utilities with a `dark:` variant;
- literal hex, RGB, HSL, or OKLCH colors in renderable source;
- local CSS or SCSS imports from renderable source;
- inline `color`, `background`, `borderColor`, `fill`, or `stroke` values.

Status styling uses Tailwind aliases backed by Carbon support roles, such as support error,
warning, success, and information. Raw red, yellow, green, and blue palette utilities are
not status exceptions. Checker fixtures cover each forbidden category, variant prefixes,
important modifiers, and each exact exception form.

The Tailwind theme removes literal gray, slate, and custom color scales after their
consumers reach zero. It exposes Carbon-backed semantic aliases only. `border-subtle`
aliases the selected Carbon border role directly; it does not add a `color-mix()` opacity
decision outside Carbon.

Runtime color APIs required by charts, editors, React Flow, canvas, or other third-party
libraries must receive resolved semantic properties through a theme-aware adapter. An
unavoidable construct is allowed only as an exact checker exception with a reason and a
focused test. Carbon-component selection and composition quality remain code-review
requirements. They are verified against a component inventory in
`ui/src/app/components/ui/README.md`, not inferred by the checker.

`ui/src/styles/custom.css` is removed. Rules expressible through Carbon props or Tailwind
move to their owning components; unavoidable CodeMirror, Monaco, React Flow, flatpickr,
and other vendor selectors move to focused files under `ui/src/styles/externals/` with
exact checker exceptions and tests. The two existing component CSS modules are converted
to Carbon and Tailwind and removed. Component CSS receives no permanent exception.

The long migration uses `ui/scripts/theme-violation-baseline.json`. The checker records
counts by exact path and rule category, fails any increase or newly violating file, and
requires a migrated path to remain at zero. Its baseline-update command can only lower or
remove counts after a reviewed migration; it has no increase or skip option. Once all
entries reach zero, the baseline is removed and every rule becomes an unconditional error.

Single source of truth means one selected config and one resolved Carbon property set per
built deployment. The repository's development, production, and Docker configs are
separate deployment profiles and may intentionally differ. Contract tests verify that
each is complete and valid; they do not require profiles to have identical values.

## Contracts and Ownership

- `ui/src/theme/` owns both public schema versions, parsing, version 1 compatibility,
  the machine-readable contract, active-mode state, document application, and structured
  issues.
- `ui/src/index.tsx` owns fail-closed bootstrap and neutral user output.
- `ui/src/configs/` and `docker/ui/config.*.json` own their deployable version 2 values.
- `ui/src/styles/carbon.scss` owns Carbon preset installation and brand-role overrides.
- `ui/src/styles/tailwind.css` owns Tailwind aliases to Carbon semantic properties.
- `ui/src/styles/externals/` owns tested third-party selector adapters after
  `ui/src/styles/custom.css` is removed.
- `ui/src/app/components/ui/` owns shared primitives, the component inventory, and Carbon
  composition rules.
- `ui/scripts/check-theme-contract.mjs` owns mechanically enforceable styling rules and
  exact third-party exceptions. Its checked-in baseline owns the temporary migration
  debt and may only ratchet downward.
- `ui/THEMING.md` owns operator and contributor guidance after implementation ships.

The public UI configuration contract changes from schema version 1 to schema version 2.
Existing version 1 deployments remain compatible through the parser. No backend API,
authentication, authorization, database, persistent-data, or cross-service contract
changes.

## Alternatives Rejected

- Keep version 1 with hardwired Carbon white and g100 bases. Rejected because background
  selection would remain outside the deployment theme contract.
- Configure a complete application palette. Rejected because it duplicates Carbon's
  semantic system, omits internal component states over time, and expands the public
  schema without a verified need.
- Expose raw `--cds-*` names in deployable configuration. Rejected because it couples the
  public contract to Carbon package internals and makes upgrades a tenant migration.
- Calculate a complete palette from one primary brand value. Rejected because reliable
  contrast and state relationships cannot be inferred for arbitrary brands.
- Keep component-level Tailwind palettes and light/dark pairs. Rejected because values
  remain distributed across render paths and cannot be validated centrally.
- Replace Tailwind with component CSS. Rejected because it adds an authoring path without
  a current need and conflicts with the established utility-class interface.

## Consequences and Risks

| Area                         | Impact or reason for N/A                                                                                                                                                                                                                                                                                                                                                                                                 |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Compatibility                | Version 1 preserves its current computed light and dark behavior. Version 2 dark primary actions intentionally use the configured brand ramp. Old bundles and version 2 configs must be rolled back together because old bundles do not understand the new schema.                                                                                                                                                       |
| Failure and recovery         | Invalid version 2 themes stop before analytics and React, render fixed neutral text, and apply no tenant values. Operators recover by correcting the selected config or rolling back the bundle and config together. Broken assets retain alternative text and do not change the color contract.                                                                                                                         |
| Security and privacy         | Theme values are public presentation data and must not contain secrets. Failure output never reflects rejected values. CSS values are assigned through property APIs rather than HTML interpolation. Tenant-hosted assets retain RFC 0014 privacy considerations.                                                                                                                                                        |
| Data and migration           | Repository-owned deployable JSON configs migrate to version 2. There is no database or persistent user-data migration. Stored light, dark, or system mode values remain compatible.                                                                                                                                                                                                                                      |
| Concurrency and resources    | Parsing and document application are synchronous before React rendering. No worker, connection, timer, stream, or cleanup ownership changes.                                                                                                                                                                                                                                                                             |
| Operations and observability | Build-time diagnostics identify invalid config and forbidden styling by file. Runtime configuration failure uses fixed neutral output. UI Platform owns Carbon-upgrade mapping review. Contract checks inventory every Carbon property consumed by source and Tailwind, and browser probes require each property to compute under every allowed preset. Browser and screenshot checks cover both modes on stable routes. |

## Rollout and Rollback

1. Implement the final checker rules and check in a per-path, per-category baseline before
   changing component styling. The initial reviewed inventory is approximately 139 files,
   2,114 palette occurrences, 970 `dark:` variants, 309 arbitrary color utilities, and 54
   directly consumed Carbon properties; the checker output becomes the canonical count.
2. Add version 2 parsing, fixed neutral failure output, machine-readable schema data, and
   exact version 1 compatibility fixtures.
3. Add Carbon preset selection and the version-specific brand mapping. Migrate every
   repository-owned deployable config to version 2 and review dark primary-action changes.
4. Make Tailwind semantic utilities alias Carbon properties, then migrate shared
   primitives and application shells. Remove their baseline entries when they reach zero.
5. Migrate domain components and dialogs, then product routes. Assistant work is split
   into deployment and creation, agentflow editors, conversations, and analytics before
   the remaining authentication, account, dashboard, endpoint, activity, integration,
   knowledge, preview, and workspace routes.
6. Remove both component CSS modules. Classify every `custom.css` selector, move only
   unavoidable vendor adapters to tested external files, then delete `custom.css`.
7. Disable literal Tailwind palettes after all consumers reach zero and remove the
   temporary violation baseline.
8. Inspect and accept every intentional screenshot change before updating its baseline.

Each slice has one implementation owner and exits only when its baseline entries are zero,
focused component tests pass, affected stable routes pass in both modes, reviewed visual
changes are accepted, and independent review has no unresolved critical or major finding.
File counts, ownership, and exact slice commands live in the governed task plan rather
than this durable decision record.

Stop rollout if version 1 computed colors change, default version 2 themes regress serious
accessibility checks, Carbon and Tailwind resolve a shared role differently, or a stable
route has an unexplained screenshot difference.

Rollback reverts the bundle and selected deployable config together. Version 1 support
keeps external configurations usable throughout rollout. The change has no irreversible
data or service effect.

## Verification

Acceptance criteria:

- Changing `carbonTheme` changes Carbon components, Tailwind semantic utilities, shared
  primitives, and page surfaces to the selected Carbon base.
- Changing a version 2 brand value changes every Carbon and Tailwind consumer named in its
  mapping table in both modes.
- Changing version 2 `onPrimary` changes primary and active tertiary foregrounds without
  changing computed danger-button or support foregrounds.
- Carbon and Tailwind resolve each shared role to the same computed value in both modes.
- Browser probes verify every Carbon property consumed by application source or a
  Tailwind alias computes under white, g10, g90, and g100.
- All repository deployable configs provide valid version 2 light and dark settings.
- Version 1 fixtures preserve current computed light and dark primary-button, link,
  focus, border, and tertiary values.
- Partial, empty, and missing version 1 fixtures preserve all current fallback behavior.
- Invalid configuration stops before analytics and React and renders fixed neutral output
  without tenant values.
- The bounded production scan reports no forbidden color source outside exact tested
  third-party exceptions.
- The shared component inventory records the Carbon primitive or the reason a domain
  composite is necessary, and independent review finds no unjustified replacement.
- Stable critical routes pass serious accessibility checks and reviewed screenshots in
  light and dark modes.
- Computed primary foreground and background pairs in every repository-owned config meet
  WCAG AA contrast. Accessibility baseline limits do not increase.

Focused verification includes parser and bootstrap tests, frozen version 1 fixtures,
deployable-config parity, checker fixtures for every forbidden construct, computed-style
parity between Carbon controls and Tailwind semantic utilities, Carbon-property coverage,
and contrast calculations. Browser review covers authentication sign-in, dashboard,
assistant listing and detail, assistant conversations, endpoint overview, account
settings, and preview debugger routes in light and dark modes before the full stable suite.

Required commands:

```bash
cd ui && yarn check:theme-contract
cd ui && yarn test:theme-contract
cd ui && yarn lint
cd ui && yarn checkTs
cd ui && yarn test --watchAll=false --runInBand
cd ui && yarn build
just ui-browser-test
python3 bin/validate-rfc-layout
just agent-finalize "ui/src/index.tsx,ui/src/index.test.tsx,ui/src/theme,ui/src/styles,ui/src/configs,ui/src/app,ui/e2e,ui/scripts,ui/package.json,ui/THEMING.md,docker/ui,plans/0021-ui-semantic-theme-contract.md,rfcs/0021-ui-semantic-theme-contract.md"
```

## Review and Approval

- Challenger: Independent Codex plan challenger (`/root/rfc_challenge`) and external
  review supplied by the repository owner
- Outcome: Approved
- Resolved findings: Exact version 2 schema and Carbon ownership; explicit version 1 and
  failure contracts; selector-scoped `onPrimary`; machine-readable schema ownership;
  property namespace rules; bounded enforcement; migration ratchet; legacy CSS removal;
  computed-style, contrast, and Carbon-upgrade verification; partial RFC supersession.
- Open questions: None
- Approval reference: Repository owner approval in conversation on 2026-11-06
