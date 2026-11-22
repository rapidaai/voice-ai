export const INTERFACE_TOUR_STORAGE_KEY = 'rpd::__ui_tour';

const COMPLETED_STATUS = 'completed';
const LEGACY_TOUR_ID = 'dashboard';

type InterfaceTourState = Record<string, unknown>;

function readInterfaceTourState(): InterfaceTourState {
  try {
    const stored = window.localStorage.getItem(INTERFACE_TOUR_STORAGE_KEY);
    if (!stored) return {};
    if (stored === COMPLETED_STATUS) {
      return { [LEGACY_TOUR_ID]: COMPLETED_STATUS };
    }

    const parsed: unknown = JSON.parse(stored);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? (parsed as InterfaceTourState)
      : {};
  } catch {
    return {};
  }
}

export function isInterfaceTourCompleted(tourId: string) {
  return readInterfaceTourState()[tourId] === COMPLETED_STATUS;
}

export function completeInterfaceTour(tourId: string) {
  window.localStorage.setItem(
    INTERFACE_TOUR_STORAGE_KEY,
    JSON.stringify({
      ...readInterfaceTourState(),
      [tourId]: COMPLETED_STATUS,
    }),
  );
}
