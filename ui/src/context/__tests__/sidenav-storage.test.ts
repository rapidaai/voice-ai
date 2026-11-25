import {
  getSidenavExpanded,
  setSidenavExpanded,
} from '@/context/sidenav-storage';

describe('sidenav storage', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('preserves other sidenav preferences when one changes', () => {
    window.localStorage.setItem(
      'rpd::__ui_sidenav',
      JSON.stringify({ main: false, assistant: true }),
    );

    setSidenavExpanded('endpoint', false);

    expect(
      JSON.parse(window.localStorage.getItem('rpd::__ui_sidenav')!),
    ).toEqual({
      main: false,
      assistant: true,
      endpoint: false,
    });
    expect(getSidenavExpanded('endpoint')).toBe(false);
  });

  it('does not block navigation when browser storage refuses writes', () => {
    jest.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
      throw new DOMException('Storage quota exceeded', 'QuotaExceededError');
    });

    expect(() => setSidenavExpanded('assistant', false)).not.toThrow();
  });
});
