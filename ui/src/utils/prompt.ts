import { TextChatCompletePrompt, TextPrompt, Variable } from '@rapidaai/react';

export const ChatCompletePrompt = (promptConfig: {
  prompt: { role: string; content: string }[];
  variables: { name: string; type: string; defaultvalue: string }[];
}): TextChatCompletePrompt => {
  const prompt = new TextChatCompletePrompt();
  promptConfig.prompt.forEach(x => {
    const tp = new TextPrompt();
    tp.setRole(x.role);
    tp.setContent(x.content);
    prompt.addPrompt(tp);
  });

  promptConfig.variables.forEach(x => {
    const v = new Variable();
    v.setType(x.type);
    v.setDefaultvalue(x.defaultvalue);
    v.setName(x.name);
    prompt.addPromptvariables(v);
  });

  return prompt;
};

/**
 *
 * @param cfg
 * @returns
 */
export const Prompt = (
  cfg: TextChatCompletePrompt,
): {
  prompt: { role: string; content: string }[];
  variables: { name: string; type: string; defaultvalue: string }[];
} => {
  return {
    prompt: cfg.getPromptList().map(tp => ({
      role: tp.getRole(),
      content: tp.getContent(),
    })),
    variables: cfg.getPromptvariablesList().map(v => ({
      name: v.getName(),
      type: v.getType(),
      defaultvalue: v.getDefaultvalue(),
    })),
  };
};
