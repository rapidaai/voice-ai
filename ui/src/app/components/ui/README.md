# UI Component Structure

This directory is split by component responsibility:

- `primitives`: reusable design-system wrappers and small base components.
- `composites`: reusable UI assemblies built from primitives or Carbon components.
- `feedback`: notifications, empty states, indicators, and loading states.
- `editor`: code, JSON, and markdown editing or rendering components.
- `table`: table-specific components and cells.

Prefer adding new shared controls to the smallest matching group. Keep domain-specific
behavior in `components/domain` or the owning page folder.

Complex composites can use folder modules when they need private model logic or
subcomponents. For example, `composites/query-search` owns its parser/model module,
component, tests, and public index together.

New shared imports from page and domain code should prefer the group indexes when
practical:

- `@/app/components/ui/primitives`
- `@/app/components/ui/composites`
- `@/app/components/ui/feedback`
- `@/app/components/ui/editor`
- `@/app/components/ui/table`

Inside `components/ui`, prefer exact sibling module imports over group indexes. This
keeps primitive, editor, feedback, and table modules from creating barrel cycles or
loading unrelated browser-only dependencies during focused tests.

Components that import browser-only packages or app configuration should stay
direct-import only and should not be exported from group barrels. Current examples are
`composites/audio-player`, `editor/markdown-viewer`,
`primitives/buttons/arrow-button`, and `primitives/buttons/social-button-group`.
Dialog modules with store dependencies, such as `dialogs/shared/create-tag-modal`,
follow the same direct-import rule.

Avoid importing production page or domain code from a specific shared component file
unless the module has a default export, owns a browser-only dependency, or a focused
test mock needs a file-level boundary.

## Carbon Component Inventory

The shared component layer owns the Carbon boundary. A wrapper preserves the application
API while delegating interaction, accessibility, and state behavior to the named Carbon
primitive.

| Application component or family | Carbon implementation | Verification |
| --- | --- | --- |
| Buttons, icon buttons, and loading buttons | `Button` and `ButtonSkeleton` | `primitives/__tests__/button.test.tsx` |
| Breadcrumb, dropdown, list box, select, slider, toggle, tabs, tooltip, overflow menu, pagination, checkbox, text input, and text area wrappers | Matching Carbon component | Focused tests under `primitives/__tests__` |
| Static cards and tiles | `Tile` and `SkeletonPlaceholder` | Domain component tests and stable browser journeys |
| Radio selection cards | `RadioTile` | `primitives/__tests__/checkbox-card.test.tsx` |
| Modal and form wrappers | Carbon modal, form, stack, and field components | Primitive tests plus route form tests |
| Table, table head, row, header, body, and cell wrappers | Carbon table primitives | Tests under `table/__tests__` and `table/table.stories.tsx` |
| Notifications, toast, loading, empty state, and status indicators | Carbon notification, loading, button, and preview indicator components | Tests under `feedback/__tests__` |
| Date filter, tag input, table toolbar filter, and menu composites | Carbon date, tag, checkbox, popover, button, and overflow-menu primitives | Tests under `composites/__tests__` |
| Code and JSON editors | Carbon actions around Monaco or CodeMirror editor surfaces | `editor/__tests__/editor-components.test.tsx` and the registered editor adapter |

Carbon does not supply every product interaction. The following retained composites use
native semantics only where substituting a Carbon component would remove required
behavior. Their colors still resolve through semantic Tailwind aliases backed by Carbon
properties.

| Retained composite | Why a direct Carbon replacement is not complete | Behavioral coverage |
| --- | --- | --- |
| `composites/query-search` | This is a parser-driven inline token editor with editable filter chips, field and logic pickers, and an anchored calendar. Carbon supplies surrounding controls but has no query-grammar editor with its focus and token-editing contract. | `composites/query-search/__tests__/query-search-model.test.ts` and `composites/query-search/__tests__/query-search.test.tsx` |
| `primitives/ScalableTextarea` | Carbon `TextArea` owns ordinary text areas. The prompt editor variant needs content-driven height, an inline action slot, and no nested field wrapper. | `primitives/__tests__/textarea.test.tsx` |
| `table/TableForm` | It is a compact form grid whose cells contain Carbon fields. It is not tabular data, so DataTable selection, sorting, and row behavior would be misleading. | `table/__tests__/table-form.test.tsx` |
| `table/ScrollableResizableTable` | It composes Carbon table primitives but retains the application column-resize contract, which Carbon DataTable does not provide. | `table/__tests__/data-table.test.tsx` |
| `composites/CreateFullPage*` | It composes Carbon progress and step primitives with route-owned bodies, validation, and action regions required by the existing create flows. | `composites/__tests__/create-full-page.test.tsx` and create-flow route tests |
| Agentflow canvas editors | React Flow owns drag, selection, connection, and canvas focus behavior. Carbon fields and actions are composed inside nodes and panels; native canvas triggers remain inside the React Flow boundary. | Agentflow unit tests and create-agentflow Playwright journeys |
| Audio and voice controls | Browser media elements and the voice SDK own playback, capture, and push-to-talk lifecycles. Carbon buttons are used for ordinary surrounding actions. | Audio-player and voice-agent unit tests plus preview-agent browser journeys |
| `dialogs/shared/RightSideModal` resize handle | Carbon has no drawer-width resize control. The native button is a pointer-driven separator handle with a fixed accessible name; all ordinary modal actions remain Carbon controls. | `dialogs/shared/__tests__/right-side-modal.test.tsx` |
| Hidden form and upload inputs | React Hook Form requires hidden identity inputs, and react-dropzone owns its hidden file-input props. These elements have no visual or direct interaction surface; visible fields and actions use Carbon. | Workspace and project modal tests plus the create-knowledge-document flow test |

Provider and route code must use these shared wrappers for ordinary controls. A new native
control requires a row in this inventory with a Carbon gap, an owner, and focused behavior
coverage. Visual preference is not a sufficient exception.

## Carbon Typography

Use the `Text` primitive from `primitives/text.tsx` for Carbon typesets. The UI bundles
IBM Plex Sans and IBM Plex Mono locally, and `typographyClassNames` is the source of
truth for the corresponding utility classes.

```tsx
<Text as="h1" variant="heading-06">
  Log in
</Text>
<Text as="p" variant="body-02" className="text-muted">
  Don’t have an account?
</Text>
```

Use `className` for color, spacing, and layout. Do not override the type size, line
height, weight, or letter spacing of a selected Carbon variant. Fixed heading variants
are intentionally not responsive.

All application-authored copy in pages and layouts, including titles, subtitles,
labels, helper text, headers, and footers, must use `Text` with a canonical Carbon
variant. When a component cannot render `Text`, compose its class name from
`typographyClassNames`. Do not introduce raw `text-*`, `font-*`, `leading-*`, or
`tracking-*` typography utilities in page or layout code.

Primary action labels must use the shared `PrimaryButton`. It renders labels through
`Text` with the productive `body-compact-01` typeset. The selected Carbon preset and
configured brand ramp own default, hover, active, and foreground colors. Do not
restyle primary button typography or colors in page code.

Use Carbon components for controls, fields, navigation, tables, notifications,
overlays, and loading states. Domain composites may combine Carbon primitives while
preserving their states, keyboard behavior, type roles, spacing, and icon conventions.
Use Tailwind semantic classes for component layout and color roles. Do not add a
component stylesheet, literal palette class, structural `dark:` color pair, arbitrary
color, or direct Carbon property expression.

### Utility styles

Productive utility styles end in `-01`; expressive utility styles end in `-02`.

| Variant          | Font          | Size / line height | Weight | Letter spacing | Type set   | Use                                               |
| ---------------- | ------------- | ------------------ | ------ | -------------- | ---------- | ------------------------------------------------- |
| `code-01`        | IBM Plex Mono | 12px / 16px        | 400    | 0.32px         | Productive | Inline snippets and small code elements           |
| `code-02`        | IBM Plex Mono | 14px / 20px        | 400    | 0.32px         | Expressive | Large snippets and larger code elements           |
| `label-01`       | IBM Plex Sans | 12px / 16px        | 400    | 0.32px         | Productive | Field labels, errors, and captions, not body copy |
| `label-02`       | IBM Plex Sans | 14px / 18px        | 400    | 0.16px         | Expressive | Field labels, errors, and captions, not body copy |
| `helper-text-01` | IBM Plex Sans | 12px / 16px        | 400    | 0.32px         | Productive | Explanatory text below a field title              |
| `helper-text-02` | IBM Plex Sans | 14px / 18px        | 400    | 0.16px         | Expressive | Explanatory text below a field title              |
| `legal-01`       | IBM Plex Sans | 12px / 16px        | 400    | 0.32px         | Productive | Legal copy in product pages                       |
| `legal-02`       | IBM Plex Sans | 14px / 18px        | 400    | 0.16px         | Expressive | Legal copy in web pages                           |

### Body styles

| Variant           | Font          | Size / line height | Weight | Letter spacing | Type set   | Use                                                    |
| ----------------- | ------------- | ------------------ | ------ | -------------- | ---------- | ------------------------------------------------------ |
| `body-compact-01` | IBM Plex Sans | 14px / 18px        | 400    | 0.16px         | Productive | Short component copy up to four lines                  |
| `body-compact-02` | IBM Plex Sans | 16px / 22px        | 400    | 0              | Expressive | Short expressive copy, buttons, and links              |
| `body-01`         | IBM Plex Sans | 14px / 20px        | 400    | 0.16px         | Productive | Longer left-aligned body and structured component copy |
| `body-02`         | IBM Plex Sans | 16px / 24px        | 400    | 0              | Expressive | Longer left-aligned expressive body copy               |

### Fixed heading styles

| Variant              | Font          | Size / line height | Weight | Letter spacing | Type set   | Use                                                          |
| -------------------- | ------------- | ------------------ | ------ | -------------- | ---------- | ------------------------------------------------------------ |
| `heading-compact-01` | IBM Plex Sans | 14px / 18px        | 600    | 0.16px         | Productive | Compact component and layout headings with `body-compact-01` |
| `heading-compact-02` | IBM Plex Sans | 16px / 22px        | 600    | 0              | Expressive | Small layout headings with `body-compact-02`                 |
| `heading-01`         | IBM Plex Sans | 14px / 20px        | 600    | 0.16px         | Productive | Component and layout headings with `body-01`                 |
| `heading-02`         | IBM Plex Sans | 16px / 24px        | 600    | 0              | Expressive | Small layout headings with `body-02`                         |
| `heading-03`         | IBM Plex Sans | 20px / 28px        | 400    | 0              | Productive | Component and layout headings                                |
| `heading-04`         | IBM Plex Sans | 28px / 36px        | 400    | 0              | Productive | Layout headings                                              |
| `heading-05`         | IBM Plex Sans | 32px / 40px        | 400    | 0              | Productive | Layout headings                                              |
| `heading-06`         | IBM Plex Sans | 42px / 50px        | 300    | 0              | Productive | Large layout headings                                        |
| `heading-07`         | IBM Plex Sans | 54px / 64px        | 300    | 0              | Productive | Largest fixed layout headings                                |

The primitive uses an explicit weight of 600 for Carbon semi-bold headings because the
application’s `font-semibold` Tailwind token is intentionally mapped to IBM Plex Medium
at weight 500. Existing semantic variants such as `pageTitle` remain available for
compatibility, but they are product aliases rather than canonical Carbon typeset names.

## Stories And Tests

Use native Storybook CSF stories beside the component group they document. Small
wrappers may share a group story when the story still exposes each public state clearly;
for example, `primitives/primitives.stories.tsx` documents the primitive wrappers and
`table/table.stories.tsx` documents table composition and pagination.

Keep UI component tests in the nearest `__tests__` folder:

- `primitives/__tests__` for primitive wrappers.
- `table/__tests__` for table primitives and pagination.
- `feedback/__tests__`, `editor/__tests__`, and `composites/__tests__` for their
  matching groups.
- Folder modules, such as `composites/query-search`, may keep their own local
  `__tests__` folder beside private model code.

Run focused UI tests with:

```bash
just ui-test src/app/components/ui
```

Build Storybook before publishing UI structure changes:

```bash
just ui-storybook-build
```
