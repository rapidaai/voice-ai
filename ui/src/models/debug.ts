export enum PromptRole {
  system = 'system',
  user = 'user',
  assistant = 'assistant',
}

export type PromptVariable = {
  name: string;
  type: string;
  defaultvalue: string | null;
};
