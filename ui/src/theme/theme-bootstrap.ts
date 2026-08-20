import { assertThemeManifest } from './theme-config';
import { ThemeManifest } from './types';

export const getValidatedThemeOrRenderError = (
  value: unknown,
  rootElement: HTMLElement,
): ThemeManifest | null => {
  try {
    assertThemeManifest(value);
    return value;
  } catch (error) {
    rootElement.textContent = 'Application configuration error.';
    console.error(
      '[theme] Invalid CONFIG.theme. The application was not started.',
      error,
    );
    return null;
  }
};
