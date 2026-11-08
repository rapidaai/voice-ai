import themeContract from './theme-contract.json';

export type ThemeMode = 'light' | 'dark' | 'system';

export type ResolvedThemeMode = Exclude<ThemeMode, 'system'>;

export type LightCarbonTheme = keyof typeof themeContract.carbonThemes.light;
export type DarkCarbonTheme = keyof typeof themeContract.carbonThemes.dark;
export type ThemeColorField = keyof typeof themeContract.colorFields;

export type ThemeColors = Record<ThemeColorField, string>;

export type LightThemeColors = ThemeColors & {
  carbonTheme: LightCarbonTheme;
};

export type DarkThemeColors = ThemeColors & {
  carbonTheme: DarkCarbonTheme;
};

export interface ThemeManifest {
  schemaVersion: 1 | 2;
  id: string;
  brand: {
    name: string;
    environmentLabel?: string;
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
    light: LightThemeColors;
    dark: DarkThemeColors;
  };
}

export type ThemeConfigIssueCode =
  | 'required'
  | 'invalid-type'
  | 'unsupported'
  | 'internal-reference';

export interface ThemeConfigIssue {
  path: string;
  code: ThemeConfigIssueCode;
}

export type ThemeParseResult =
  | { ok: true; theme: ThemeManifest }
  | { ok: false; issues: ThemeConfigIssue[] };
