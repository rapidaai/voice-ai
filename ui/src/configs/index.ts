/* eslint-disable import/no-mutable-exports */
import { InputVarType } from '@/models/common';
import { ConnectionConfig } from '@rapidaai/react';

export let WEB_API = 'http://dev.rapida.local';
export let ASSISTANT_API = 'http://assistant.rapida.local';
export let WEB_HOST = 'http://localhost:3000';

export let connectionConfig = new ConnectionConfig({
  assistant: 'http://assistant.rapida.local',
  web: 'http://dev.rapida.local',
  endpoint: 'http://dev.rapida.local',
});

if (process.env.NODE_ENV === 'production') {
  WEB_API = 'https://api.rapida.ai';
  ASSISTANT_API = 'https://assistant-01.rapida.ai';
  WEB_HOST = 'https://rapida.ai';
  connectionConfig = new ConnectionConfig({
    assistant: 'https://assistant-01.rapida.ai',
    web: 'https://api.rapida.ai',
    endpoint: 'https://api.rapida.ai',
  });
}

export const MAX_PROMPT_MESSAGE_LENGTH = 10;
export const MAX_VAR_KEY_LENGHT = 100;
export const DEFAULT_VALUE_MAX_LEN = 500;

export const VAR_ITEM_TEMPLATE = {
  name: '',
  type: 'string',
  defaultvalue: '',
};

export const VAR_ITEM_TEMPLATE_IN_WORKFLOW = {
  variable: '',
  label: '',
  type: InputVarType.textInput,
  max_length: DEFAULT_VALUE_MAX_LEN,
  required: true,
  options: [],
};

export const zhRegex = /^[\u4E00-\u9FA5]$/m;
export const emojiRegex = /^[\uD800-\uDBFF][\uDC00-\uDFFF]$/m;
export const emailRegex = /^[\w\.-]+@([\w-]+\.)+[\w-]{2,}$/m;
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
export const UPDATE_DATASETS_EVENT_EMITTER =
  'prompt-editor-context-block-update-datasets';
export const UPDATE_HISTORY_EVENT_EMITTER =
  'prompt-editor-history-block-update-role';

export const SUPPORTED_PROMPT_VARIABLE_TYPE = () => {
  return [
    InputVarType.textInput,
    InputVarType.paragraph,
    InputVarType.number,
    InputVarType.url,
    InputVarType.json,
  ];
};
