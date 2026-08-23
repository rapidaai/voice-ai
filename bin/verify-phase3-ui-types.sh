#!/usr/bin/env bash
set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ui_root="${repository_root}/ui"
config_path="${ui_root}/.rfc-0001-tsconfig.tmp.json"
diagnostics_path="${TMPDIR:-/tmp}/rfc-0001-ui-types.log"

cleanup() {
  rm -f "${config_path}"
}
trap cleanup EXIT

cat >"${config_path}" <<'JSON'
{
  "compilerOptions": {
    "target": "ESNext",
    "lib": ["dom", "dom.iterable", "esnext"],
    "types": ["jest", "node", "react", "react-dom"],
    "allowJs": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "strict": true,
    "forceConsistentCasingInFileNames": true,
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "noFallthroughCasesInSwitch": true,
    "jsx": "react-jsx",
    "baseUrl": ".",
    "paths": {"@/*": ["./src/*"]}
  },
  "files": [
    "src/app/pages/assistant/listing/__tests__/assistant-table-ux.test.tsx",
    "src/app/pages/assistant/listing/single-assistant.tsx",
    "src/app/pages/assistant/view/version-list/index.tsx",
    "src/app/pages/endpoint/listing/__tests__/endpoint-table-ux.test.tsx",
    "src/app/pages/endpoint/listing/index.tsx",
    "src/app/pages/endpoint/listing/single-endpoint.tsx",
    "src/app/pages/endpoint/view/version-list/index.tsx",
    "src/hooks/use-assistant-provider-page-store.ts",
    "src/hooks/use-endpoint-page-store.ts",
    "src/hooks/use-knowledge-activity-log-page-store.ts",
    "src/types/types.endpoint.ts",
    "src/utils/audit-actor.test.ts",
    "src/utils/audit-actor.ts"
  ]
}
JSON

compiler_status=0
(
  cd "${TMPDIR:-/tmp}"
  npm exec --yes --package=typescript@5.9.3 -- tsc -p "${config_path}"
) >"${diagnostics_path}" 2>&1 || compiler_status=$?

if [ "${compiler_status}" -eq 0 ]; then
  echo "Phase 3 UI TypeScript verification passed."
  exit 0
fi

if rg -n '^error TS|node_modules/.*error TS' "${diagnostics_path}"; then
  echo "TypeScript failed before reliable changed-source verification." >&2
  exit 1
fi

changed_source_pattern='ui/src/(app/pages/(assistant|endpoint)/|hooks/use-(assistant-provider|endpoint|knowledge-activity-log)-page-store|types/types\.endpoint|utils/audit-actor)'
if rg -n "${changed_source_pattern}" "${diagnostics_path}"; then
  echo "Phase 3 UI source files contain TypeScript diagnostics." >&2
  exit 1
fi

diagnostic_count="$(wc -l <"${diagnostics_path}" | tr -d ' ')"
echo "Phase 3 UI source files type-check; ${diagnostic_count} unrelated repository diagnostics remain in ${diagnostics_path}."
