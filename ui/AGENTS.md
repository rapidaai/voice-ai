# UI Agent Rules

The repository-level `AGENTS.md` remains authoritative. These rules add UI-specific requirements
for work under this directory.

- Read `DEVELOPMENT.md` and `TESTING.md` before changing UI behavior or test infrastructure.
- Keep React, Vite, Yarn, Jest, Testing Library, Storybook, and Playwright as the established stack.
- Define loading, empty, error, permission, responsive, light-theme, and dark-theme behavior when
  they apply to the changed experience.
- Add or update a focused Jest test for component and state behavior. Include a happy path and the
  regression or failure path that motivated the change.
- Add or update Playwright coverage when a route, journey, accessibility outcome, or rendered layout
  changes.
- Every enabled route must have explicit smoke, accessibility, and screenshot decisions in
  `e2e/route-manifest.js`. A deferred decision includes a concrete reason.
- Accessibility debt is recorded by journey, rule, selector, and maximum node count. Do not increase
  a limit to make a new regression pass.
- Update screenshot baselines intentionally and review every changed image in both color modes.
- Run the closest test first, then the UI verification gate documented in `TESTING.md`.
- Do not edit files outside `ui/` for a UI-only task.
