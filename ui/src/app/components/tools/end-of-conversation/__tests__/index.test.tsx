import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { ConfigureEndOfConversation } from '../index';
import {
  GetEndOfConversationDefaultOptions,
  ValidateEndOfConversationDefaultOptions,
} from '../constant';

jest.mock('../../common', () => ({
  ToolDefinitionForm: ({
    documentationTitle,
    documentationPath,
    toolDefinition,
  }: any) => (
    <div>
      <div data-testid="doc-title">{documentationTitle}</div>
      <div data-testid="doc-path">{documentationPath}</div>
      <div data-testid="tool-name">{toolDefinition?.name}</div>
    </div>
  ),
}));

describe('end-of-conversation tool', () => {
  it('returns empty defaults and no validation error', () => {
    expect(GetEndOfConversationDefaultOptions([])).toEqual([]);
    expect(ValidateEndOfConversationDefaultOptions([])).toBeUndefined();
  });

  it('renders ToolDefinitionForm with end-of-conversation docs', () => {
    render(
      <ConfigureEndOfConversation
        inputClass=""
        parameters={[]}
        onParameterChange={jest.fn()}
        toolDefinition={{
          name: 'end_conversation',
          description: 'desc',
          parameters: '{}',
        }}
        onChangeToolDefinition={jest.fn()}
      />,
    );

    expect(screen.getByTestId('doc-title')).toHaveTextContent(
      'Know more about supported End of Conversation behavior',
    );
    expect(screen.getByTestId('doc-path')).toHaveTextContent(
      '/assistants/tools/add-end-of-conversation-tool',
    );
    expect(screen.getByTestId('tool-name')).toHaveTextContent(
      'end_conversation',
    );
  });
});
