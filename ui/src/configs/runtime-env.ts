type RuntimeEnvValue = string | boolean | undefined;

export interface RuntimeEnv {
  mode: string;
  gaMeasurementId?: string;
}

const readProcessEnv = (): NodeJS.ProcessEnv => process.env;

const readString = (value: RuntimeEnvValue) =>
  typeof value === 'string' && value.trim() ? value.trim() : undefined;

export const getRuntimeEnv = (): RuntimeEnv => {
  const processEnv = readProcessEnv();

  return {
    mode:
      readString(processEnv?.VITE_MODE) ??
      readString(processEnv?.MODE) ??
      readString(processEnv?.NODE_ENV) ??
      'development',
    gaMeasurementId:
      readString(processEnv?.VITE_GA_MEASUREMENT_ID) ??
      readString(processEnv?.REACT_APP_GA_MEASUREMENT_ID),
  };
};
