import {
  assertThemeManifest,
  ThemeManifestValidationError,
} from './theme-config';
import { ThemeManifest } from './types';

export const getValidatedThemeOrRenderError = (
  value: unknown,
  rootElement: HTMLElement,
): ThemeManifest | null => {
  try {
    assertThemeManifest(value);
    return value;
  } catch (error) {
    const diagnostic =
      error instanceof ThemeManifestValidationError
        ? error.diagnostics[0]
        : 'CONFIG.theme could not be validated.';
    rootElement.textContent = `Application configuration error: ${diagnostic}`;
    console.error(
      '[theme] Invalid CONFIG.theme. The application was not started.',
      error,
    );
    return null;
  }
};
