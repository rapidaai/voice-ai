import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { THEME_STORAGE_KEY } from './theme-config';
import { safeStorageGet, safeStorageSet } from './theme-storage';
import {
  ResolvedThemeMode,
  ThemeColorField,
  ThemeManifest,
  ThemeMode,
} from './types';
import themeContract from './theme-contract.json';

interface ThemeContextValue {
  theme: ThemeManifest;
  mode: ThemeMode;
  resolvedMode: ResolvedThemeMode;
  setMode: (mode: ThemeMode) => void;
  toggleMode: () => void;
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined);

const getSystemMode = (): ResolvedThemeMode =>
  window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';

const getStoredMode = (theme: ThemeManifest): ThemeMode => {
  if (!theme.allowModeSelection) return theme.defaultMode;

  const storedMode = safeStorageGet(THEME_STORAGE_KEY);
  return storedMode === 'light' ||
    storedMode === 'dark' ||
    storedMode === 'system'
    ? storedMode
    : theme.defaultMode;
};

const isThemeMode = (value: string | null): value is ThemeMode =>
  value === 'light' || value === 'dark' || value === 'system';

const carbonThemeClasses = [
  ...Object.keys(themeContract.carbonThemes.light),
  ...Object.keys(themeContract.carbonThemes.dark),
].map(theme => `cds--${theme}`);

export const resolveThemeMode = (
  mode: ThemeMode,
  systemMode: ResolvedThemeMode,
): ResolvedThemeMode => (mode === 'system' ? systemMode : mode);

export const applyThemeToDocument = (
  theme: ThemeManifest,
  resolvedMode: ResolvedThemeMode,
) => {
  const root = document.documentElement;
  const colors = theme.colors[resolvedMode];

  root.setAttribute(themeContract.documentAttributes.brand, theme.id);
  root.setAttribute(themeContract.documentAttributes.colorMode, resolvedMode);
  root.setAttribute(
    themeContract.documentAttributes.carbonTheme,
    colors.carbonTheme,
  );
  root.classList.remove(...carbonThemeClasses);
  root.classList.add(`cds--${colors.carbonTheme}`);
  root.setAttribute(themeContract.documentAttributes.themeReady, 'true');
  root.setAttribute(
    themeContract.documentAttributes.themeSchema,
    String(theme.schemaVersion),
  );
  root.style.colorScheme = resolvedMode;
  for (const field of Object.keys(
    themeContract.colorFields,
  ) as ThemeColorField[]) {
    root.style.setProperty(
      themeContract.colorFields[field].property,
      colors[field],
    );
  }

  if (theme.brand.favicon) {
    const icons = document.querySelectorAll<HTMLLinkElement>(
      "link[rel~='icon'], link[rel='apple-touch-icon']",
    );
    if (icons.length === 0) {
      const favicon = document.createElement('link');
      favicon.rel = 'icon';
      favicon.href = theme.brand.favicon;
      document.head.appendChild(favicon);
    } else {
      icons.forEach(icon => {
        icon.href = theme.brand.favicon as string;
      });
    }
  }

  let themeColor = document.querySelector<HTMLMetaElement>(
    "meta[name='theme-color']",
  );
  if (!themeColor) {
    themeColor = document.createElement('meta');
    themeColor.name = 'theme-color';
    document.head.appendChild(themeColor);
  }
  themeColor.content = colors.primary;
};

export const getInitialThemeState = (theme: ThemeManifest) => {
  const mode = getStoredMode(theme);
  const systemMode = getSystemMode();
  return { mode, resolvedMode: resolveThemeMode(mode, systemMode) };
};

export const ThemeProvider: React.FC<{
  theme: ThemeManifest;
  children: React.ReactNode;
}> = ({ theme, children }) => {
  const [mode, setModeState] = useState<ThemeMode>(() => getStoredMode(theme));
  const [systemMode, setSystemMode] =
    useState<ResolvedThemeMode>(getSystemMode);
  const previousAllowModeSelection = useRef(theme.allowModeSelection);
  const resolvedMode = resolveThemeMode(mode, systemMode);

  useEffect(() => {
    if (!theme.allowModeSelection) {
      setModeState(theme.defaultMode);
    } else if (!previousAllowModeSelection.current) {
      setModeState(getStoredMode(theme));
    }
    previousAllowModeSelection.current = theme.allowModeSelection;
  }, [theme.allowModeSelection, theme.defaultMode]);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = (event: MediaQueryListEvent) =>
      setSystemMode(event.matches ? 'dark' : 'light');

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  useEffect(() => {
    if (!theme.allowModeSelection) return undefined;

    const handleStorage = (event: StorageEvent) => {
      if (event.key === THEME_STORAGE_KEY && isThemeMode(event.newValue)) {
        setModeState(event.newValue);
      }
    };

    window.addEventListener('storage', handleStorage);
    return () => window.removeEventListener('storage', handleStorage);
  }, [theme.allowModeSelection]);

  useLayoutEffect(() => {
    applyThemeToDocument(theme, resolvedMode);
  }, [resolvedMode, theme]);

  const setMode = useCallback(
    (nextMode: ThemeMode) => {
      if (!theme.allowModeSelection) return;
      safeStorageSet(THEME_STORAGE_KEY, nextMode);
      setModeState(nextMode);
    },
    [theme.allowModeSelection],
  );

  const toggleMode = useCallback(() => {
    setMode(resolvedMode === 'dark' ? 'light' : 'dark');
  }, [resolvedMode, setMode]);

  const value = useMemo(
    () => ({ theme, mode, resolvedMode, setMode, toggleMode }),
    [mode, resolvedMode, setMode, theme, toggleMode],
  );

  return (
    <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
  );
};

export const useTheme = (): ThemeContextValue => {
  const context = useContext(ThemeContext);
  if (!context) throw new Error('useTheme must be used within ThemeProvider');
  return context;
};
