# UI Semantic Theme Contract Plan

## Context

- Task: Make the selected UI theme and resolved Carbon properties the single source of
  truth for application colors. Use Carbon components for supported patterns and Tailwind
  semantic classes for component styling.
- Verified current behavior: The version 1 config supplies four brand colors. Carbon white
  and g100 supply remaining roles. Production components also use literal palettes,
  arbitrary color utilities, direct Carbon expressions, structural dark variants, two CSS
  modules, and a global `custom.css` file.
- Problem: Backgrounds, actions, content, borders, and states have multiple owners, so
  visually equivalent components can diverge across modes and implementation styles.
- Assumptions: IBM Carbon remains the design-system standard; Tailwind remains the utility
  interface; repository-owned configs move to version 2; external version 1 configs retain
  current output.
- Open questions: None.

The reviewed inventory reported 139 violating files, 2,114 palette occurrences, 970
`dark:` variants, 309 arbitrary color utilities, and 54 directly consumed Carbon
properties. Scan patterns currently differ, so the first implementation stage produces
the canonical checked-in baseline before any migration claim is made.

## Acceptance

- Acceptance criteria:
  - Every repository-owned config satisfies the machine-readable version 2 contract.
  - Version 1 partial, empty, and missing structures retain current fallback behavior and
    computed light and dark output.
  - Carbon controls and Tailwind semantic utilities compute the same shared roles under
    white, g10, g90, and g100.
  - Version 2 brand values control the RFC mapping without changing danger or support
    foregrounds through `onPrimary`.
  - Invalid version 2 configuration stops before analytics and React and renders fixed
    neutral output without tenant values.
  - The violation baseline never increases, each completed slice reaches zero, and the
    final repository has no baseline.
  - Components use Carbon when it supplies the pattern. Domain composites record why a
    Carbon composition is required.
  - `custom.css` and both component CSS modules are removed. Tested third-party adapters
    are the only styling exceptions.
  - Required unit, contract, build, accessibility, computed-style, contrast, screenshot,
    finalization, and independent review gates pass.
- Non-goals:
  - Backend branding, APIs, authentication, authorization, data, or deployment protocols.
  - Runtime tenant switching, provider logos, repository identity, or package identity.
  - Product workflow redesign or public configuration of raw Carbon properties.
  - New UI dependencies unless an accepted follow-up proves an existing dependency cannot
    satisfy a required Carbon pattern.

## RFC

- Reserved path: `rfcs/0021-ui-semantic-theme-contract.md`
- Plan artifact: `plans/0021-ui-semantic-theme-contract.md`
- Sidecar JSON: Prohibited by `bin/validate-rfc-layout`; lifecycle evidence remains in the
  RFC, plan, task, and review records.
- RFC author: Primary Codex agent.
- Exact reviewed RFC SHA-256:
  `fed7ad892925f98e474af3376de9209866388c6642f86b1fc77cf81a796b2d66`.
- Confirmation gate / receipt: RFC accepted after independent challenge and repository-owner
  approval in conversation on 2026-11-06.

## Scope and Ownership

| Path or module                                                                                        | Owner                            | Responsibility                                                                                   |
| ----------------------------------------------------------------------------------------------------- | -------------------------------- | ------------------------------------------------------------------------------------------------ |
| `rfcs/0021-ui-semantic-theme-contract.md`                                                             | RFC author                       | Durable decision, challenge findings, and approval evidence                                      |
| `plans/0021-ui-semantic-theme-contract.md`                                                            | Task planner                     | File scope, migration stages, owners, ratchet, and exact verification                            |
| `ui/src/theme/**`                                                                                     | Theme contract implementer       | Machine-readable schema, types, parser, compatibility, mode state, and document properties       |
| `ui/src/index.tsx`, `ui/src/index.test.tsx`                                                           | Bootstrap implementer            | Fail-closed decision, neutral output, and analytics/React exclusion                              |
| `ui/src/configs/**`, `docker/ui/config.*.json`                                                        | Deployable-config implementer    | Complete version 2 repository profiles and parity fixtures                                       |
| `ui/src/styles/carbon.scss`, `ui/src/styles/tailwind.css`, `ui/src/styles/externals/**`               | Theme adapter implementer        | Carbon presets, brand mapping, semantic aliases, and tested vendor adapters                      |
| `ui/jest.config.cjs`                                                                                 | UI test compatibility implementer | Jest ESM transformation required by the approved Carbon compatibility update                     |
| `ui/scripts/check-theme-contract.mjs`, adjacent tests, and `ui/scripts/theme-violation-baseline.json` | Contract-check implementer       | Machine schema consumption, forbidden-pattern rules, exact exceptions, and downward-only ratchet |
| `ui/src/app/components/ui/**`, `ui/src/app/components/layout/**`                                      | Component-system implementer     | Carbon primitives, semantic classes, inventory, and shared-shell migration                       |
| `ui/src/app/components/domain/**`, `ui/src/app/components/dialogs/**`                                 | Domain-component implementer     | Carbon composition and removal of component CSS modules                                          |
| `ui/src/app/pages/assistant/**`                                                                       | Assistant UI implementer         | Deployment, creation, agentflow, conversation, and analytics slices                              |
| Remaining `ui/src/app/pages/**` and route consumers                                                   | Application UI implementer       | Route-sliced migration and regression coverage                                                   |
| `ui/e2e/**`                                                                                           | Browser-verification implementer | Computed styles, contrast, stable-route screenshots, and accessibility                           |
| `ui/THEMING.md`, `ui/src/app/components/ui/README.md`                                                 | UI documentation implementer     | Operator contract and Carbon component inventory                                                 |

One implementer owns each row in a task dispatch. Theme contract, adapter, and checker
stages run serially because their write sets and acceptance evidence overlap. Route slices
may run separately only after the foundation settles and their files do not overlap.

- Allowed paths:
  - The paths and modules listed in the ownership table.
  - `ui/package.json` and `ui/yarn.lock` for repository-owned verification scripts and the
    Carbon compatibility update. The repository owner separately requested and approved
    the Carbon update on 2026-11-06. The approved versions are `@carbon/react` 1.120.0 and
    `@carbon/styles` 1.119.0.
  - `ui/.stylelintrc` only to recognize the Tailwind directives used by the semantic
    utility contract.
  - `ui/jest.config.cjs` only for the `temporal-polyfill`, `temporal-spec`, and
    `temporal-utils` ESM transformation required by the owner-approved Carbon update.
- Out-of-scope paths:
  - `api/**`, `pkg/**`, `cmd/**`, `protos/**`, backend configuration, and deployment
    protocols.
  - Generated CSS under `ui/src/styles/generated/**`; update it only through the existing
    build command.
  - Provider catalog names and assets, repository branding, licensing, and legal identity.

## Migration Slices and Ratchet

The initial file counts below are the supplied review inventory. Stage 1 replaces them
with checker-produced counts per exact path and rule category.

| Slice                                 |                               Initial violating files | Owner                                           | Exit criteria                                                                            |
| ------------------------------------- | ----------------------------------------------------: | ----------------------------------------------- | ---------------------------------------------------------------------------------------- |
| Theme, config, adapters, and checker  |                                                   N/A | Theme contract and adapter implementers         | Schema, compatibility, failure, parity, coverage, and ratchet tests pass                 |
| Shared UI primitives and layout       |                               25 plus layout findings | Component-system implementer                    | Owned baseline entries are zero; Carbon inventory complete; shell browser checks pass    |
| Domain components and dialogs         |                            29 (18 domain, 11 dialogs) | Domain-component implementer                    | Owned entries are zero; both CSS modules removed; focused tests pass                     |
| Assistant deployment and creation     | Count frozen by Stage 1 within the 50 assistant files | Assistant UI implementer                        | Owned entries are zero; create, edit, and deployment journeys pass both modes            |
| Agentflow editors                     | Count frozen by Stage 1 within the 50 assistant files | Assistant UI implementer                        | Both editor paths are zero; graph/editor exceptions are exact and tested                 |
| Assistant conversations and analytics | Count frozen by Stage 1 within the 50 assistant files | Assistant UI implementer                        | Conversation and analytics paths are zero; detail and screenshot journeys pass           |
| Remaining application routes          |                                                    35 | Application UI implementer                      | Every remaining path is zero; affected stable routes pass both modes                     |
| Legacy global CSS                     |  About 105 `custom.css` selectors and two CSS modules | Theme adapter and domain-component implementers | Expressible rules migrated; vendor adapters tested; `custom.css` and modules deleted     |
| Final strict mode                     |                                                  Zero | Contract-check implementer                      | Baseline deleted; palettes disabled; every forbidden construct is an unconditional error |

`ui/scripts/theme-violation-baseline.json` records counts by path and rule. The default
checker fails a count increase or a newly violating file. The update command can only
lower counts or remove entries. It has no increase, skip, or alternate-checker option.

## Principle Decisions

- KISS / smallest complete solution: Configure a Carbon preset plus the existing four
  brand values. Do not create a parallel full application palette.
- YAGNI / rejected alternatives: No raw Carbon configuration map, calculated palette,
  component CSS system, runtime theme service, or new dependency.
- Single source of truth: The selected config chooses the base and brand ramp; the resolved
  Carbon property set plus selector-scoped `--brand-*` inputs is the only runtime source.
  Tailwind contains aliases, not independent values.
- Explicit contracts and compatibility: `theme-contract.json` owns schema data consumed by
  runtime and Node tooling. Version 1 output is frozen. Version 2 is strict and fail-closed.
- Failure safety and cleanup: Invalid version 2 input stops analytics and React, exposes no
  tenant values, and shows fixed text. Invalid JSON fails the build. No runtime resource is
  created before validation.
- Security and least privilege: Theme values are public, contain no secrets, and never
  enter HTML interpolation. Version 2 rejects internal Carbon and brand variable references.
- Observability: Build diagnostics name path and rule. Browser probes cover computed
  Carbon properties. No runtime tenant value appears in failure output.
- Rollback, disablement, or migration: Roll back bundle and repository-owned version 2
  config together. External version 1 configs remain supported. Each slice is independently
  revertible after foundation changes settle.

## Implementation Stages

1. Governance gate: accept RFC 0021 and record owner approval. No source edit starts first.
2. Baseline and checker: implement all forbidden categories, exact exceptions, checker
   fixtures, Carbon-property inventory, and the downward-only baseline.
3. Contract and bootstrap: add the machine-readable schema, versioned parser, frozen
   version 1 fixtures, strict version 2 failure, and bootstrap tests.
4. Carbon and Tailwind adapters: compile allowed presets, scope brand mappings, map semantic
   utilities directly, remove extra border opacity, and add computed-style and contrast
   probes.
5. Config migration: update six repository profiles, verify role parity and both modes,
   and review intentional dark primary-action screenshots.
6. Shared system migration: convert primitives and shells, then domain components and
   dialogs. Remove both CSS modules.
7. Product slices: assistant deployment and creation, agentflow, assistant conversations
   and analytics, then remaining routes. Lower the baseline with each slice.
8. CSS closure: move unavoidable vendor selectors to tested external adapters, delete
   `custom.css`, disable literal palettes, and delete the zero baseline.
9. Full verification and independent review against one fixed candidate boundary.

## Verification

- Required test categories:
  - Machine schema, all six deployable configs, version 1 compatibility, version 2 errors.
  - Bootstrap exclusion of analytics, React, tenant attributes, properties, and text.
  - Checker fixtures for palettes, arbitrary colors, dark color variants, literals, CSS
    imports, inline colors, direct Carbon expressions, exceptions, and ratchet behavior.
  - Computed Carbon and Tailwind parity under white, g10, g90, and g100.
  - Computed WCAG contrast for repository-owned primary pairs and danger isolation.
  - Focused component success and edge coverage for every migrated slice.
  - Stable critical screenshots and serious accessibility checks in both modes.
- Exact commands:

```bash
cd ui && yarn check:theme-contract
cd ui && yarn test:theme-contract
cd ui && yarn test --watchAll=false --runInBand src/index.test.tsx src/theme src/styles src/app/components
cd ui && yarn lint
cd ui && yarn checkTs
cd ui && yarn test --watchAll=false --runInBand
cd ui && yarn build
cd ui && yarn playwright test e2e/journeys/auth/authentication.spec.js e2e/journeys/dashboard/state-screenshots.spec.js e2e/journeys/deployment/assistant-listing.spec.js e2e/journeys/deployment/assistant-detail.spec.js e2e/journeys/preview/debugger-v2.spec.js --config=playwright.config.js
just ui-browser-test
python3 bin/validate-rfc-layout
just agent-finalize "ui/src/index.tsx,ui/src/index.test.tsx,ui/src/theme,ui/src/styles,ui/src/configs,ui/src/app,ui/e2e,ui/scripts,ui/jest.config.cjs,ui/package.json,ui/yarn.lock,ui/THEMING.md,docker/ui,plans/0021-ui-semantic-theme-contract.md,rfcs/0021-ui-semantic-theme-contract.md"
```

Screenshot baselines change only after direct inspection. `e2e/a11y-baseline.js` limits
must not increase. Final verification records exact commands, exit codes, and environment
limitations.

## Discussion

- Planner: Primary Codex agent.
- Challenger: Independent Codex plan challenger and external review supplied by repository
  owner.
- Blocking concerns: Migration ratchet, property rules, legacy CSS disposition, Carbon
  upgrade coverage, verification specificity, and missing plan artifact.
- Resolutions: Recorded in RFC 0021 and this plan. Targeted re-review approved the fixed
  RFC and plan boundary with no unresolved critical or major findings. Final review also
  confirmed that `ui/jest.config.cjs` is a required compatibility adapter for the Carbon
  update requested and approved by the repository owner on 2026-11-06; its ownership,
  allowed scope, and finalization path are recorded above.
- Decision: Approved for implementation.
- Approved by: Repository owner in conversation on 2026-11-06.
