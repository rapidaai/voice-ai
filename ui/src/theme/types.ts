export type ThemeMode = 'light' | 'dark' | 'system';

export type ResolvedThemeMode = Exclude<ThemeMode, 'system'>;

export interface ThemeColors {
  primary: string;
  primaryHover: string;
  primaryActive: string;
  onPrimary: string;
}

export interface ThemeManifest {
  schemaVersion: 1;
  id: string;
  brand: {
    name: string;
    logos?: {
      full: {
        light: string;
        dark: string;
      };
      compact: {
        light: string;
        dark: string;
      };
    };
    favicon?: string;
  };
  links: {
    documentation: string;
    source: string;
    support: string;
    terms: string;
    privacy: string;
  };
  defaultMode: ThemeMode;
  allowModeSelection: boolean;
  colors: {
    light: ThemeColors;
    dark: ThemeColors;
  };
}
