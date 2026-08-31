import { FC } from 'react';
import { ConfigureToolProps, ToolDefinitionForm } from '../common';

// ============================================================================
// Main Component
// ============================================================================

export const ConfigureEndOfConversation: FC<ConfigureToolProps> = ({
  inputClass,
  toolDefinition,
  onChangeToolDefinition,
}) => (
  <>
    {toolDefinition && onChangeToolDefinition && (
      <ToolDefinitionForm
        toolDefinition={toolDefinition}
        onChangeToolDefinition={onChangeToolDefinition}
        inputClass={inputClass}
        documentationPath="/assistants/tools/add-end-of-conversation-tool"
        documentationTitle="Know more about supported End of Conversation behavior"
      />
    )}
  </>
);
