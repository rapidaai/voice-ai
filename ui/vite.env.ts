import { loadEnv } from 'vite';

export const getClientProcessEnv = (mode: string) => {
  const fileEnv = loadEnv(mode, process.cwd(), '');
  const readEnv = (name: string) => process.env[name] ?? fileEnv[name];
  const appMode = readEnv('VITE_MODE') ?? mode;

  return {
    MODE: appMode,
    NODE_ENV: appMode,
    VITE_MODE: appMode,
    VITE_GA_MEASUREMENT_ID: readEnv('VITE_GA_MEASUREMENT_ID'),
    REACT_APP_GA_MEASUREMENT_ID: readEnv('REACT_APP_GA_MEASUREMENT_ID'),
  };
};
