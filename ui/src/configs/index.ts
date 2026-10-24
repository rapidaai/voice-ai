/* eslint-disable import/no-mutable-exports */
import { InputVarType } from '@/models/common';
import { ConnectionConfig } from '@rapidaai/react';
import type { ThemeManifest } from '@/theme/types';
import devConfig from './config.development.json';
import prodConfig from './config.production.json';
import { getRuntimeEnv } from './runtime-env';

export interface SentryAnalyticsConfig {
  dsn: string;
  tracePropagationTargets: string[];
}

export interface WorkspaceConfig {
  domain: string;
  authentication: {
    signIn: {
      providers: Record<'password' | 'google' | 'linkedin' | 'github', boolean>;
    };
    signUp: {
      enable: boolean;
      providers: Record<'password' | 'google' | 'linkedin' | 'github', boolean>;
    };
    passwordRules?: {
      minLength?: number;
      requireUppercase?: boolean;
      requireLowercase?: boolean;
      requireNumber?: boolean;
      requireSpecialChar?: boolean;
    };
  };
  features?: {
    knowledge?: boolean;
    telemetry?: boolean;
  };
}

export interface RapidaConfig {
  connection: {
    media: string;
    assistant: string;
    web: string;
    endpoint: string;
    sip?: string;
    socket?: string;
  };
  deployment: {
    webWidgetScriptUrl: string;
  };
  analytics?: SentryAnalyticsConfig;
  workspace: WorkspaceConfig;
  theme: ThemeManifest;
}

type ConfigSource = Omit<RapidaConfig, 'deployment' | 'theme'> & {
  deployment?: Partial<RapidaConfig['deployment']>;
  theme: unknown;
};

const WEB_WIDGET_SCRIPT_PATH = '/public/scripts/app.min.js';

const readConfigString = (value: unknown): string | undefined =>
  typeof value === 'string' && value.trim().length > 0
    ? value.trim()
    : undefined;

const withoutTrailingSlash = (value: string): string =>
  value.endsWith('/') ? value.slice(0, -1) : value;

const isLocalWorkspaceDomain = (domain: string): boolean =>
  domain === 'localhost' ||
  domain === '127.0.0.1' ||
  domain === '::1' ||
  domain.endsWith('.localhost');

export const withConfigDefaults = (config: ConfigSource): RapidaConfig => {
  const domain = readConfigString(config.workspace?.domain);
  const webOrigin = readConfigString(config.connection.web);
  const webWidgetScriptUrl =
    readConfigString(config.deployment?.webWidgetScriptUrl) ??
    (domain && isLocalWorkspaceDomain(domain) && webOrigin
      ? `${withoutTrailingSlash(webOrigin)}${WEB_WIDGET_SCRIPT_PATH}`
      : domain
        ? `https://cdn.${domain}${WEB_WIDGET_SCRIPT_PATH}`
        : WEB_WIDGET_SCRIPT_PATH);

  return {
    ...config,
    deployment: {
      webWidgetScriptUrl,
    },
    theme: config.theme as ThemeManifest,
  };
};
const developmentConfig = withConfigDefaults(devConfig as ConfigSource);
const productionConfig = withConfigDefaults(prodConfig as ConfigSource);

export const getConfig = (environment = getRuntimeEnv().mode): RapidaConfig =>
  environment === 'production' ? productionConfig : developmentConfig;

export const CONFIG = getConfig();
export const connectionConfig = new ConnectionConfig(CONFIG.connection);

//
export const MAX_PROMPT_MESSAGE_LENGTH = 10;
export const MAX_VAR_KEY_LENGHT = 100;
export const VAR_ITEM_TEMPLATE = {
  name: '',
  type: 'string',
  defaultvalue: '',
};

export const zhRegex = /^[\u4E00-\u9FA5]$/m;
const MAX_ZN_VAR_NAME_LENGHT = 8;
const MAX_EN_VAR_VALUE_LENGHT = 30;
export const getMaxVarNameLength = (value: string) => {
  if (zhRegex.test(value)) return MAX_ZN_VAR_NAME_LENGHT;
  return MAX_EN_VAR_VALUE_LENGHT;
};

export const CONTEXT_PLACEHOLDER_TEXT = '{{#context#}}';
export const HISTORY_PLACEHOLDER_TEXT = '{{#histories#}}';
export const QUERY_PLACEHOLDER_TEXT = '{{#query#}}';
export const PRE_PROMPT_PLACEHOLDER_TEXT = '{{#pre_prompt#}}';

export const SUPPORTED_PROMPT_VARIABLE_TYPE = () => {
  return [
    InputVarType.textInput,
    InputVarType.paragraph,
    InputVarType.number,
    InputVarType.url,
    InputVarType.json,
  ];
};
