import developmentConfig from '@/configs/config.development.json';
import { getValidatedThemeOrRenderError } from '@/theme/theme-bootstrap';

describe('theme bootstrap', () => {
  afterEach(() => jest.restoreAllMocks());

  it('returns the configured theme without mutating the root', () => {
    const root = document.createElement('div');

    expect(getValidatedThemeOrRenderError(developmentConfig.theme, root)).toBe(
      developmentConfig.theme,
    );
    expect(root).toBeEmptyDOMElement();
  });

  it('fails closed before rendering when configuration is invalid', () => {
    const root = document.createElement('div');
    const consoleError = jest.spyOn(console, 'error').mockImplementation();

    expect(getValidatedThemeOrRenderError({ id: 'invalid' }, root)).toBeNull();
    expect(root).toHaveTextContent('Application configuration error.');
    expect(document.documentElement).not.toHaveAttribute('data-brand');
    expect(consoleError).toHaveBeenCalledWith(
      '[theme] Invalid CONFIG.theme. The application was not started.',
      expect.any(Error),
    );
  });
});
