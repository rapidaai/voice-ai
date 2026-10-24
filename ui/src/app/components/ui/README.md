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
`Text` with the productive `body-compact-01` typeset and owns the white-on-primary
color contract for default, hover, and active states. Dark mode uses an accessible
darker ramp derived from the configured primary color. Do not restyle primary button
typography or colors in page code.

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
