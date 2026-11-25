export const SIDENAV_STORAGE_KEY = 'rpd::__ui_sidenav';

export type SidenavId = 'main' | 'assistant' | 'endpoint';

const LEGACY_SIDENAV_ID: SidenavId = 'main';

type SidenavState = Record<string, unknown>;

function readSidenavState(): SidenavState {
  try {
    const stored = window.localStorage.getItem(SIDENAV_STORAGE_KEY);
    if (!stored) return {};
    if (stored === 'true' || stored === 'false') {
      return { [LEGACY_SIDENAV_ID]: stored === 'true' };
    }

    const parsed: unknown = JSON.parse(stored);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? (parsed as SidenavState)
      : {};
  } catch {
    return {};
  }
}

export function getSidenavExpanded(sidenavId: SidenavId) {
  const expanded = readSidenavState()[sidenavId];
  return typeof expanded === 'boolean' ? expanded : true;
}

export function setSidenavExpanded(sidenavId: SidenavId, expanded: boolean) {
  try {
    window.localStorage.setItem(
      SIDENAV_STORAGE_KEY,
      JSON.stringify({
        ...readSidenavState(),
        [sidenavId]: expanded,
      }),
    );
  } catch {
    return;
  }
}
