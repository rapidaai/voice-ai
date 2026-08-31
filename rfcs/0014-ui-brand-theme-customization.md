# RFC 0014: UI Brand and Theme Customization

- Status: Draft
- Owner: UI Platform
- Created: 2026-08-31
- Updated: 2026-08-31
- Reviewers: UI reviewer, product owner

## Summary

Complete the existing UI brand and theme configuration so web UI surfaces use
the configured brand name, logo assets, favicon, documentation links, legal
links, support link, and primary colors from `CONFIG.theme`.

The current theming work already provides a strong foundation. This RFC limits
the remaining work to UI runtime and UI-shipped static assets. Backend emails,
email templates, API contracts, Go module paths, package names, and licensing
identity are not part of this proposal.

## Context

The UI already loads `CONFIG.theme` from the selected deployable config and
wraps React with the theme provider in `ui/src/index.tsx`. The provider applies
the active brand id, color mode, favicon, browser theme color, and CSS brand
tokens in `ui/src/theme/theme-provider.tsx`.

The main shell is partially converted. Header, sidebar, footer, onboarding, and
helmet components already read values through `useTheme()` for visible brand
name, logo, or product links. The deployable config files under
`ui/src/configs/` and `docker/ui/` already contain a root `theme` object with
brand, link, mode, and color fields.

The remaining UI gaps are places that bypass `CONFIG.theme` and still render
Rapida-specific values. Verified examples include:

- `ui/src/app/pages/preview-agent/voice-agent/voice-agent.tsx`: preview header
  imports and renders `RapidaIcon` and `RapidaTextIcon` directly.
- `ui/src/app/pages/preview-agent/voice-agent/text/conversations/index.tsx`:
  assistant chat messages render a Rapida icon and the display name `Rapida`.
- UI pages and components still contain hardcoded Rapida documentation URLs,
  support email placeholders, product copy, static legal copy, public AI metadata,
  and Rapida-hosted dashboard or tool assets.

The theme contract checker already verifies single-source theme loading, shell
semantic color usage, required deployable theme config presence, and removal of
legacy dark-mode sources. It does not yet enforce the broader brand contract for
new UI code.

## Goals

- Make `CONFIG.theme` the source of truth for user-visible UI brand identity.
- Ensure preview call and preview chat render configured brand assets and brand
  name.
- Ensure UI navigation, page metadata, auth links, legal/support links, and
  documentation links use configured theme links or explicit neutral fallback
  behavior.
- Remove direct Rapida logo component usage from tenant-visible UI surfaces when
  configured logos exist.
- Add automated checks that catch new hardcoded Rapida brand literals in UI code
  outside an explicit allowlist.
- Preserve existing default Rapida appearance for the stock config files.

## Non-Goals

- Backend email subjects, email sender identity, and email body templates.
- Go module paths, generated clients, protobuf package names, npm package names,
  import paths, and internal metadata keys such as `rapida.credential_id`.
- Product or legal rewording beyond removing unintended UI hardcoding.
- Provider catalog branding, provider names, and third-party provider logos.
- Runtime tenant switching without rebuilding the selected UI bundle.
- Authentication, authorization, API, database, or deployment contract changes.

## Scope and Ownership

### Allowed Paths

- `ui/src/theme/` - UI theme contract and runtime theme ownership.
- `ui/src/configs/` - development and production UI config defaults.
- `docker/ui/config.*.json` - Docker edition UI config defaults.
- `ui/src/app/components/` - shared UI components and branded UI surfaces.
- `ui/src/app/pages/` - UI pages that render brand names, docs links, support
  links, static legal pages, dashboard content, or preview chat.
- `ui/src/app/routes/` - route-level branded layout surfaces.
- `ui/public/` - UI-shipped public metadata and default brand assets.
- `ui/scripts/check-theme-contract.mjs` and related tests - UI contract checks.
- `ui/THEMING.md` - UI branding and theme documentation.

### Out-of-Scope Paths

- `api/**`
- `pkg/clients/external/emailer/**`
- `pkg/configs/emailer_config.go`
- `cmd/**`
- `protos/**`
- `pkg/**` except files explicitly listed above, which are none.
- `README.md`, `LICENSE.md`, `SECURITY.md`, and other repository identity docs.
- Package manager lockfiles and generated artifacts.

## Proposed Design

Keep the existing build-time model. The selected UI config remains bundled into
the web app, and there is no runtime brand fetch, tenant cache, or HTML bootstrap
script.

Use the existing `theme` object without changing `schemaVersion` for the first
implementation. The current fields are sufficient for the requested UI scope:

- `theme.brand.name`
- `theme.brand.logos.full.light`
- `theme.brand.logos.full.dark`
- `theme.brand.logos.compact.light`
- `theme.brand.logos.compact.dark`
- `theme.brand.favicon`
- `theme.links.documentation`
- `theme.links.source`
- `theme.links.support`
- `theme.links.terms`
- `theme.links.privacy`
- `theme.colors.light`
- `theme.colors.dark`

Introduce a small shared branded logo component only if it removes repeated
logo selection code across header, sidebar, onboarding, and preview surfaces.
The component should accept a full or compact variant, read `useTheme()`, choose
the asset for the resolved mode, and fall back to text from `theme.brand.name`
when no logo is configured. It must not fall back to embedded Rapida SVGs on
tenant-visible surfaces.

Update preview call and chat surfaces to use the same brand contract:

- Preview header uses the configured full logo for the active mode.
- Assistant chat avatar uses the configured compact logo for the active mode.
- Assistant chat display name uses `theme.brand.name`.
- Avatar colors use existing semantic brand tokens instead of a fixed blue when
  the configured compact logo is absent.

Replace hardcoded documentation URLs in UI components with links derived from
`theme.links.documentation`. Components that need section-specific docs should
accept a relative documentation path and build the absolute URL from the theme
documentation root, or should receive a fully configured URL from a theme-aware
caller. Existing `DocNoticeBlock` behavior can remain as the display component,
but ownership of branded URL construction must be explicit.

Auth and legal navigation should use `theme.links.terms`,
`theme.links.privacy`, and `theme.links.support`. The built-in Rapida legal
pages may remain only as default deployment pages when the selected config
points to them. White-label navigation must not point users to Rapida legal copy
unless the selected config explicitly chooses those links.

Static UI-shipped metadata such as `ui/public/LLMs.txt` must not be a second
hand-maintained brand source. It should either be made neutral or generated from
the selected build config. The preferred first implementation is neutral content
unless product explicitly requires branded generated metadata.

Extend `ui/scripts/check-theme-contract.mjs` to catch new hardcoded UI brand
literals. The check should scan UI source and selected public metadata, ignore
default config values, ignore tests where literal assertions are intentional,
and ignore technical identifiers that are not user-visible branding. The
allowlist must be explicit and small.

## Contracts and Compatibility

- No backend API contract changes.
- No database or persistent-data changes.
- No UI config schema version change is required for the initial implementation.
- Existing default configs continue to render Rapida AI by setting
  `theme.brand.name` and Rapida asset paths in config.
- Missing configured logos fall back to configured brand text, not embedded
  Rapida marks.
- Invalid or empty theme fields continue to use the existing UI fallback
  behavior from the theme config layer.

## Failure and Recovery

- If a configured logo URL is broken, the browser shows the configured alt text
  from `theme.brand.name`; the shell remains usable.
- If `theme.links.documentation`, `theme.links.terms`, `theme.links.privacy`,
  `theme.links.source`, or `theme.links.support` are empty or malformed, the
  existing config fallback behavior applies.
- If the brand contract check reports hardcoded literals, implementation stops
  until the source is moved to config, changed to neutral copy, or added to an
  explicit allowlist with justification.

## Security and Privacy

- Theme values are public UI presentation config and must not contain secrets.
- External links opened from themed URLs should preserve existing safe link
  behavior, including `rel="noopener"` where a new tab is used.
- The change must not weaken authentication, authorization, project isolation,
  organization isolation, or credential handling.
- Tenant-specific asset URLs are loaded by the browser. Operators own the
  privacy and tracking implications of configured asset hosts.

## Observability

No new backend observability is required. UI diagnostics remain build-time and
test-time checks:

- Theme contract diagnostics identify the file and hardcoded brand violation.
- Existing test failures identify surfaces that no longer honor configured
  brand values.

## Data and Migration

None. This proposal changes bundled UI behavior and deployable config defaults
only. No persistent data changes are required.

## Rollout

1. Update UI components and pages to consume the theme contract.
2. Update or add focused UI tests for preview header, preview chat, auth links,
   and any shared branded logo component.
3. Extend the theme contract checker and its tests.
4. Update `ui/THEMING.md` with the completed UI brand contract and the explicit
   exclusions.
5. Run targeted UI verification, full UI validation, and repository finalization.

Rollout stops if the default Rapida config no longer renders the existing Rapida
appearance, if custom brand tests still show Rapida text or marks, or if the
theme contract checker cannot distinguish user-visible branding from technical
identifiers.

## Rollback

Revert the UI component, public metadata, config, documentation, and contract
checker changes. Because this RFC does not change backend contracts or data,
rollback is a normal UI bundle rollback with no migration step.

## Alternatives Considered

- Add a second runtime brand manifest. Rejected because `CONFIG.theme` already
  exists, is documented, and avoids another source of truth.
- Rebrand backend email and repository identity in the same RFC. Rejected
  because the current request is UI-only and email requires a separate backend
  configuration owner.
- Keep Rapida SVG components as universal fallback. Rejected for tenant-visible
  UI because a missing logo would leak the default brand instead of using the
  configured brand name.
- Replace every technical identifier containing Rapida. Rejected because imports,
  package names, metadata keys, and SDK names are compatibility contracts rather
  than UI brand presentation.

## Testing and Verification

- `cd ui && yarn check:theme-contract`
- `cd ui && yarn test:theme-contract`
- `cd ui && yarn lint`
- `cd ui && yarn checkTs`
- `cd ui && yarn test --watchAll=false --runInBand`
- `cd ui && yarn build`
- `just agent-finalize "ui/src,ui/public,ui/scripts,ui/THEMING.md,docker/ui"`

Focused UI tests must include:

- Preview header renders configured logo and brand text fallback.
- Preview chat assistant avatar and assistant display name use configured theme
  values.
- Auth links use configured terms, privacy, and support links.
- Default Rapida configs still render the stock brand.
- Custom test theme does not render Rapida text or embedded Rapida logo
  components on tenant-visible UI surfaces.

## Acceptance Criteria

- [ ] `CONFIG.theme` is the only source for tenant-visible UI brand name, logo,
  favicon, primary color, docs links, legal links, and support link.
- [ ] Preview call and preview chat pages no longer import or render Rapida logo
  components directly.
- [ ] Assistant preview chat uses `theme.brand.name` for the assistant-side
  display name.
- [ ] Missing logo config falls back to configured brand text, not Rapida SVGs.
- [ ] UI navigation and auth surfaces use configured legal and support links.
- [ ] Hardcoded Rapida brand literals in UI source are either removed or listed
  in a documented allowlist with a non-branding justification.
- [ ] UI contract tests fail when new tenant-visible hardcoded Rapida brand
  literals are added.
- [ ] Default deployable configs retain the current Rapida AI appearance.
- [ ] Backend email branding remains unchanged and explicitly out of scope.

## Open Questions

- Should `ui/public/LLMs.txt` become neutral, be removed, or be generated from
  the selected UI config?
- Should built-in static legal pages remain accessible for direct URLs in
  white-label builds, or should those routes redirect to configured legal links?
- Should UI package metadata icons in `ui/package.json` be included in a later
  desktop packaging scope?

## Challenge Resolution

Pending independent plan challenge.

## Artifact Index

- `jsons/plan.json` - Initial UI-only implementation plan. Draft.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-31 | Initial UI-only RFC draft | UI Platform | `jsons/plan.json` |
