# RFC 0012: Tag and Package Successful Main Branch Builds

- Status: Accepted
- Owner: Voice AI maintainers
- Created: 2026-08-29
- Updated: 2026-08-29
- Reviewers: Independent repository maintainer

## Summary

Keep the deprecated `document-api` implementation in the repository, but exclude it from
all CI, packaging, and release service matrices. Make `just ci` install or isolate the
small tooling dependencies it owns instead of relying on undeclared host packages. After
`Continuous Integration` succeeds for a push to `main`, create one immutable repository
tag named `build-YYYYMMDD-<short-merge-commit-sha>`, package every active service, and
attach the packages plus checksums to a GitHub prerelease for that tag.

## Context

The active CI stack contains `web-api`, `integration-api`, `endpoint-api`,
`assistant-api`, and `ui`. The deprecated Python `document-api` remains in the source tree
for a later product removal, but it must not be built, tested, packaged, or started by the
current CI flow.

The existing `Package Artifacts` workflow runs for manually-created `v*` tags and packages
three Go services plus the UI. It does not create a build tag after a successful merge and
does not package the native-runtime `assistant-api` service. The existing Docker publish
workflow is separate and continues to publish Docker Hub images.

A maintainer also reproduced local `just ci` failures caused by undeclared Python modules,
an interactive stdin read in the changed-test hook, commitlint module resolution, and
direct host `shellcheck` calls. These failures make local CI depend on workstation state.

## Goals

- Keep `document-api` out of CI, packaging, and release execution without deleting its
  source or runtime files.
- Make `just ci` run from the documented prerequisites without globally installed
  `jsonschema`, `PyYAML`, commitlint packages, or `shellcheck`.
- Avoid waiting for terminal stdin when changed files were not piped to repository hooks.
- Create one deterministic tag after successful CI on a push to `main`.
- Package all active services from the exact tagged merge commit.
- Attach packages and a checksum manifest to a GitHub prerelease using the same tag.
- Make reruns idempotent when the expected tag already points to the same commit.

## Non-Goals

- Do not delete or modify the `api/document-api`, `docker/document-api`, protobuf, SDK,
  knowledge API, or local knowledge-profile implementation.
- Do not publish packages for `document-api`.
- Do not change Docker Hub image publishing, production deployment, or marketplace image
  workflows.
- Do not infer semantic-version major, minor, or patch changes from commit messages.
- Do not automatically promote a build prerelease to a production release.

## Scope and Ownership

### Allowed Paths

- `.github/workflows/package.yml`: reusable package and GitHub prerelease workflow.
- `.github/workflows/tag-and-package-services.yml`: post-CI tag orchestration.
- `.github/workflows/workflow-lint.yml`: workflow path coverage if required.
- `.github/actions/package-assistant-service/`: native assistant package action.
- `.github/actions/package-go-service/action.yml`: tag-aligned artifact naming if required.
- `.github/actions/package-directory/action.yml`: tag-aligned artifact naming if required.
- `just/ci.just`: local CI dependency wrappers and service-boundary check.
- `just/ci-commitlint.sh`: isolated commitlint module resolution.
- `just/ci-contracts.sh`: shared shellcheck fallback.
- `just/ci-stack.sh`: shared shellcheck fallback and CI service boundary.
- `just/ci-python.sh`: isolated pinned Python tooling environment.
- `just/requirements-ci.txt`: pinned Python tooling dependencies.
- `just/ci-service-boundaries.sh`: CI matrix regression check.
- `just/shellcheck.sh`: reusable shellcheck entry point.
- `.codex/hooks/validate_changed_tests.py`: non-blocking terminal input behavior.
- `.claude/hooks/validate_changed_tests.py`: mirrored hook behavior.
- `.codex/README.md`: local tooling prerequisites.
- `.claude/README.md`: mirrored local tooling prerequisites.
- `README.md`: release automation documentation.
- `rfcs/0012-main-merge-build-packages/`: governed workflow artifacts.
- `rfcs/0012-main-merge-build-packages.md`: this RFC.

### Out-of-Scope Paths

- `api/document-api/**`
- `docker/document-api/**`
- `docker-compose.knowledge.yml`
- `env/document.yaml`
- `protos/**`
- `sdks/**`
- `api/assistant-api/**`
- `api/web-api/**`
- `docker-compose.yml`
- Production deployment manifests and marketplace build definitions

## Proposed Design

### CI Service Boundary

Add a small CI check that fails when active CI, packaging, or release matrices reference
`document-api`. The check covers only CI-facing files and does not inspect local knowledge
development configuration. Existing source and compatibility code remain untouched.

### Self-Contained Local CI

Use a repository script to create a cache-scoped Python virtual environment and install
pinned `jsonschema` and `PyYAML` versions before running Python-based validation. Reuse
`just/shellcheck.sh` anywhere CI scripts currently invoke `shellcheck` directly. Export the
custom npm prefix through `NODE_PATH` when running commitlint. Treat an interactive stdin
stream as empty in both mirrored changed-test hooks, while preserving piped JSON and
`HOOK_CHANGED_FILES` behavior.

The cache directory is outside the repository and can be deleted without affecting source
state. No global package installation is required.

### Merge Tagging

Add `Tag and Package Services`, triggered by completion of `Continuous Integration` for
the `main` branch. It runs only when the triggering event was a push and the CI conclusion
was successful. It checks out `github.event.workflow_run.head_sha` and derives:

`build-<UTC commit date as YYYYMMDD>-<7-character merge commit SHA>`

The workflow creates an annotated Git tag at that exact SHA. If the tag already exists and
points to the same commit, a rerun reuses it. If it points elsewhere, the workflow fails
without moving or deleting the tag.

### Packages and Prerelease

Make `Package Artifacts` reusable so the tag workflow can package the exact tagged commit.
Retain manual and `v*` tag execution. Package the active services as follows:

- `web-api`, `integration-api`, and `endpoint-api`: Linux amd64 binary archives.
- `assistant-api`: compressed Docker runtime image archive because its executable depends
  on native libraries and model assets supplied by its runtime image.
- `ui`: compiled static build archive.

Artifact filenames include the build tag. A final job downloads all packages, generates
`SHA256SUMS`, and creates a GitHub prerelease for the build tag. It uploads the packages and
checksum manifest as release assets. The workflow uses only the repository token with
`contents: write`; build jobs retain read-only permissions.

## Contracts and Compatibility

- Existing source, protobuf, SDK, and local runtime behavior for `document-api` remains
  unchanged.
- Active CI supports exactly `web-api`, `integration-api`, `endpoint-api`,
  `assistant-api`, and `ui`.
- Build tags are repository-wide build identifiers, not semantic-version releases.
- Every package and checksum is built from the commit referenced by its build tag.
- Existing manually-created `v*` package behavior remains available.
- Existing Docker Hub image tags and publishing behavior remain unchanged.

## Failure and Recovery

- Failed or cancelled CI creates no tag and no prerelease.
- Tag creation fails safely if the generated name points to another commit.
- Packaging failure leaves the immutable tag but creates no successful prerelease; rerun
  reuses the tag and retries packaging.
- Release creation is idempotent and updates only the prerelease for the same tag.
- Missing package output or checksum generation fails the release job.
- Local dependency installation failures report the failing tool and cache path.

## Security and Privacy

- The tagging workflow accepts only successful push runs on `main`, not pull-request code.
- Checkout uses the exact trusted merge SHA from the completed CI run.
- Only the tag job and prerelease job receive `contents: write`.
- No package registry credentials or new repository secrets are introduced.
- Release assets contain build output only and no CI credentials or local configuration.

## Observability

- The tagging workflow summary records CI run, source SHA, build tag, package results, and
  prerelease URL.
- Package jobs use service-specific names and fail when expected output is missing.
- `SHA256SUMS` gives operators an integrity check for every release asset.

## Data and Migration

None.

## Rollout

Merge the workflow changes to `main`. The first successful `Continuous Integration` run
on the merge commit creates the first `build-*` tag and prerelease. Verify that the tag,
five service packages, and checksum manifest reference the same commit before using the
artifacts in a downstream release.

## Rollback

Disable or revert `tag-and-package-services.yml`. Delete an incorrect build prerelease and
its matching build tag only after confirming no downstream release consumes it. Reverting
the local CI wrappers restores the previous host-dependent behavior but does not affect
production services.

## Alternatives Considered

- Automatically increment semantic versions: rejected because merge automation cannot
  reliably infer release intent.
- Use a full commit SHA in the tag: rejected because the owner requested a short merge SHA.
- Trigger packaging from the pushed tag alone: rejected because tags created with the
  repository token do not reliably start another workflow. The orchestrator invokes the
  reusable package workflow directly.
- Publish all packages as Docker images: rejected because this request targets GitHub
  prerelease assets and the existing Docker Hub publication remains separate.
- Delete `document-api`: rejected because the clarified scope is CI-only removal.

## Testing and Verification

- `just ci-check`
- `env -u NODE_PATH just ci-commitlint`
- Run `just ci-check` with host `shellcheck`, `jsonschema`, and `PyYAML` unavailable.
- Run both mirrored changed-test hooks from a terminal without piped stdin and verify they
  return promptly.
- `actionlint`
- Validate package workflow dispatch for a test ref without creating a release.
- `just ci`
- `make agent-finalize CHANGED_FILES="comma,separated,paths"`

## Acceptance Criteria

- [ ] CI-facing service matrices and scripts contain no `document-api` service.
- [ ] The deprecated service source and non-CI local knowledge profile remain unchanged.
- [ ] `just ci` does not require globally installed Python modules, commitlint, or
  shellcheck.
- [ ] Interactive changed-test hooks do not wait for EOF.
- [ ] A successful push CI run on `main` creates exactly one deterministic `build-*` tag.
- [ ] The GitHub prerelease contains packages for all five active services and
  `SHA256SUMS`.
- [ ] Every package is produced from the exact tagged merge commit.
- [ ] Failed CI and pull-request CI runs create no tags or releases.
- [ ] Workflow reruns do not move an existing tag or create conflicting releases.
- [ ] Local `just ci` and repository finalization pass.

## Open Questions

None.

## Challenge Resolution

Pending independent review of these exact RFC and plan bytes.

## Artifact Index

- `jsons/plan.json`: proposed implementation and verification contract.

## Decision Log

| Date | Decision | Owner | Evidence |
| --- | --- | --- | --- |
| 2026-08-29 | Keep document-api source and remove it only from CI scope | Repository owner | User clarification |
| 2026-08-29 | Use build date plus short merge SHA tags | Repository owner | User confirmation |
| 2026-08-29 | Attach packages to a GitHub prerelease | Repository owner | User accepted recommendation |
