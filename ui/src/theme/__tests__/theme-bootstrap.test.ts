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
    const invalidTheme = {
      ...developmentConfig.theme,
      colors: {
        ...developmentConfig.theme.colors,
        dark: {
          ...developmentConfig.theme.colors.dark,
          primaryHover: '#7ccdf7l',
        },
      },
    };

    expect(getValidatedThemeOrRenderError(invalidTheme, root)).toBeNull();
    expect(root).toHaveTextContent(
      'Application configuration error: CONFIG.theme.colors.dark.primaryHover must be a 6-digit hexadecimal color such as #0f62fe.',
    );
    expect(root).not.toHaveTextContent('#7ccdf7l');
    expect(document.documentElement).not.toHaveAttribute('data-brand');
    expect(consoleError).toHaveBeenCalledWith(
      '[theme] Invalid CONFIG.theme. The application was not started.',
      expect.any(Error),
    );
  });

  it('renders a field-specific WCAG contrast diagnostic', () => {
    const root = document.createElement('div');
    jest.spyOn(console, 'error').mockImplementation();
    const inaccessibleTheme = {
      ...developmentConfig.theme,
      colors: {
        ...developmentConfig.theme.colors,
        light: {
          ...developmentConfig.theme.colors.light,
          primary: '#3186df',
        },
      },
    };

    expect(getValidatedThemeOrRenderError(inaccessibleTheme, root)).toBeNull();
    expect(root).toHaveTextContent(
      'Application configuration error: CONFIG.theme.colors.light.primary must have at least 4.5:1 contrast with the corresponding onPrimary color.',
    );
    expect(root).not.toHaveTextContent('#3186df');
  });
});
