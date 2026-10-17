# UI Development

Follow the repository's Fast, Standard, or Governed lifecycle. This guide defines the UI-specific
work inside those phases.

## Before implementation

1. Read the component, its closest tests, the route manifest, and any nearby Storybook stories.
2. State the user-visible behavior and the states affected by the change.
3. Select the smallest test layer that can prove each behavior.
4. For visual work, identify expected behavior in light and dark modes before editing code.

## Test selection

| Change                             | Required evidence                                     |
| ---------------------------------- | ----------------------------------------------------- |
| Pure logic, hook, store, or client | Focused Jest test                                     |
| Component behavior                 | Testing Library happy path and regression path        |
| Route rendering or navigation      | Playwright smoke journey                              |
| Accessibility behavior             | Playwright axe journey in light and dark modes        |
| Intentional visual behavior        | Reviewed Playwright snapshots in light and dark modes |
| Provider configuration             | Loader, default, and runtime-parity tests             |
| Build or test tooling              | Focused test for the checker or script                |

## Implementation loop

1. Add or update a test that demonstrates the missing behavior.
2. Confirm that the focused test fails for the expected reason.
3. Make the smallest production change that satisfies the behavior.
4. Run the focused test until it passes.
5. Run the UI verification gate.
6. Review the complete UI diff for unrelated changes and generated artifacts.

## Definition of done

- The intended behavior and important failure path have automated coverage.
- Route coverage decisions remain complete.
- Accessibility limits have not increased.
- Visual changes have reviewed light and dark snapshots.
- The production build and bundle budgets pass.
- The UI verification summary contains no failed stage.
- Every changed path is inside `ui/` when the task has an UI-only boundary.

## Build tooling

Vite owns the development and production application builds. Jest remains the unit and component
test runner, and Storybook uses its Vite framework. The production output stays in `build/` so the
existing image and deployment contract remains unchanged.
