import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import {
  BuildinTool,
  GetDefaultToolConfigIfInvalid,
  GetDefaultToolDefintion,
  ValidateToolDefaultOptions,
} from '../tool-registry';

const createMetadata = (key: string, value: string): Metadata => {
  const m = new Metadata();
  m.setKey(key);
  m.setValue(value);
  return m;
};

jest.mock('@/configs', () => ({
  CONFIG: {
    workspace: {
      features: {
        knowledge: true,
      },
    },
  },
}));

jest.mock('@carbon/icons-react', () => ({
  Information: () => <span data-testid="info-icon">i</span>,
}));

jest.mock('@carbon/react', () => ({
  Dropdown: ({ id, items, selectedItem, onChange, itemToString }: any) => (
    <select
      data-testid={id}
      value={selectedItem?.code || ''}
      onChange={e =>
        onChange({
          selectedItem: items.find((item: any) => item.code === e.target.value),
        })
      }
    >
      {items.map((item: any) => (
        <option key={item.code} value={item.code}>
          {itemToString(item)}
        </option>
      ))}
    </select>
  ),
  Select: ({ id, value, onChange, children }: any) => (
    <select data-testid={id} value={value} onChange={onChange}>
      {children}
    </select>
  ),
  Button: ({
    children,
    hasIconOnly: _hasIconOnly,
    iconDescription,
    kind: _kind,
    onClick,
    renderIcon: _renderIcon,
    size: _size,
    ...props
  }: any) => (
    <button aria-label={iconDescription} onClick={onClick} {...props}>
      {children}
    </button>
  ),
  SelectItem: ({ value, text }: any) => <option value={value}>{text}</option>,
  Table: ({ children }: any) => <table>{children}</table>,
  TableBody: ({ children }: any) => <tbody>{children}</tbody>,
  TableCell: ({ children, colSpan }: any) => (
    <td colSpan={colSpan}>{children}</td>
  ),
  TableContainer: ({ children }: any) => <section>{children}</section>,
  TableHead: ({ children }: any) => <thead>{children}</thead>,
  TableHeader: ({ children }: any) => <th>{children}</th>,
  TableRow: ({ children }: any) => <tr>{children}</tr>,
  Tooltip: ({ children }: any) => <>{children}</>,
}));

jest.mock('@/app/components/ui/primitives/input-group', () => ({
  InputGroup: ({ title, children }: any) => (
    <section>
      <h3>{title}</h3>
      {children}
    </section>
  ),
}));

jest.mock('@/app/components/domain/tools/knowledge-retrieval', () => ({
  ConfigureKnowledgeRetrieval: () => <div data-testid="tool-knowledge" />,
}));
jest.mock('@/app/components/domain/tools/knowledge-retrieval/constant', () => ({
  GetKnowledgeRetrievalDefaultOptions: (params: Metadata[]) => params,
  ValidateKnowledgeRetrievalDefaultOptions: () => undefined,
}));
jest.mock('@/app/components/domain/tools/api-request', () => ({
  ConfigureAPIRequest: () => <div data-testid="tool-api-request" />,
}));
jest.mock('@/app/components/domain/tools/api-request/constant', () => ({
  GetAPIRequestDefaultOptions: (params: Metadata[]) => params,
  ValidateAPIRequestDefaultOptions: () => undefined,
}));
jest.mock('@/app/components/domain/tools/endpoint', () => ({
  ConfigureEndpoint: () => <div data-testid="tool-endpoint" />,
}));
jest.mock('@/app/components/domain/tools/endpoint/constant', () => ({
  GetEndpointDefaultOptions: (params: Metadata[]) => params,
  ValidateEndpointDefaultOptions: () => undefined,
}));
jest.mock('@/app/components/domain/tools/end-of-conversation', () => ({
  ConfigureEndOfConversation: () => <div data-testid="tool-end-conversation" />,
}));
jest.mock('@/app/components/domain/tools/end-of-conversation/constant', () => ({
  GetEndOfConversationDefaultOptions: (params: Metadata[]) => params,
  ValidateEndOfConversationDefaultOptions: () => undefined,
}));
jest.mock('@/app/components/domain/tools/transfer-call', () => ({
  ConfigureTransferCall: () => <div data-testid="tool-transfer-call" />,
}));
jest.mock('@/app/components/domain/tools/transfer-call/constant', () => ({
  GetTransferCallDefaultOptions: (params: Metadata[]) => params,
  ValidateTransferCallDefaultOptions: () => undefined,
}));
jest.mock('@/app/components/domain/tools/mcp', () => ({
  ConfigureMCP: () => <div data-testid="tool-mcp" />,
}));
jest.mock('@/app/components/domain/tools/mcp/constant', () => ({
  GetMCPDefaultOptions: (params: Metadata[]) => params,
  ValidateMCPDefaultOptions: () => undefined,
}));

jest.mock('@/llm-tools', () => ({
  BUILDIN_TOOLS: [
    { code: 'knowledge_retrieval', name: 'Knowledge Retrieval' },
    { code: 'api_request', name: 'API request' },
    { code: 'endpoint', name: 'Endpoint (LLM Call)' },
    { code: 'end_of_conversation', name: 'End of conversation' },
    { code: 'transfer_call', name: 'Transfer call' },
    { code: 'mcp', name: 'MCP Server' },
  ],
  KnowledgeRetrievalToolDefintion: {
    name: 'knowledge_query',
    description: 'k',
    parameters: '{}',
  },
  APIRequestToolDefintion: {
    name: 'api_call',
    description: 'a',
    parameters: '{}',
  },
  EndpointToolDefintion: {
    name: 'llm_call',
    description: 'e',
    parameters: '{}',
  },
  EndOfConverstaionToolDefintion: {
    name: 'end_conversation',
    description: 'x',
    parameters: '{}',
  },
  TransferCallToolDefintion: {
    name: 'transfer_call',
    description: 't',
    parameters: '{}',
  },
}));

describe('tools index', () => {
  it('returns default and existing tool definitions correctly', () => {
    const mcpDefault = GetDefaultToolDefintion('mcp');
    expect(mcpDefault.name).toBe('mcp_tool');

    const existingMcp = GetDefaultToolDefintion('mcp', {
      name: 'custom_mcp',
      description: 'custom description',
      parameters: '{"type":"object"}',
    });
    expect(existingMcp.name).toBe('custom_mcp');

    const existingTool = GetDefaultToolDefintion('api_request', {
      name: 'custom_api',
      description: 'custom api',
      parameters: '{"type":"object"}',
    });
    expect(existingTool.name).toBe('custom_api');

    const fallback = GetDefaultToolDefintion('invalid-code');
    expect(fallback.name).toBe('llm_call');
  });

  it('normalizes tool config by injecting tool.condition metadata', () => {
    const out = GetDefaultToolConfigIfInvalid('api_request', [
      createMetadata('tool.method', 'POST'),
    ]);
    const byKey = Object.fromEntries(
      out.map(item => [item.getKey(), item.getValue()]),
    );
    expect(byKey['tool.method']).toBe('POST');
    expect(byKey['tool.condition']).toContain('"source"');
  });

  it('validates invalid code and invalid condition payload', () => {
    expect(ValidateToolDefaultOptions('invalid-code', [])).toBe(
      'Invalid tool code: invalid-code',
    );

    const err = ValidateToolDefaultOptions('api_request', [
      createMetadata('tool.condition', '{}'),
    ]);
    expect(err).toBe('Condition must be a valid JSON array.');
  });

  it('renders BuildinTool and updates source condition + action', async () => {
    const user = userEvent.setup();
    const onChangeBuildinTool = jest.fn();
    const onChangeConfig = jest.fn();

    render(
      <BuildinTool
        toolDefinition={{
          name: 'tool',
          description: 'desc',
          parameters: '{}',
        }}
        onChangeToolDefinition={jest.fn()}
        onChangeBuildinTool={onChangeBuildinTool}
        onChangeConfig={onChangeConfig}
        config={{
          code: 'knowledge_retrieval',
          parameters: [],
        }}
      />,
    );

    expect(screen.getByTestId('tool-knowledge')).toBeInTheDocument();

    await user.selectOptions(
      screen.getByTestId('tool-condition-source-value'),
      String('phone'),
    );
    const changedConfig = onChangeConfig.mock.calls[0][0];
    const condition = changedConfig.parameters.find(
      (m: Metadata) => m.getKey() === 'tool.condition',
    );
    expect(condition.getValue()).toContain('"phone"');

    await user.selectOptions(
      screen.getByTestId('tool-action-select'),
      String('api_request'),
    );
    expect(onChangeBuildinTool).toHaveBeenCalledWith('api_request');
  });

  it('supports adding another rule and setting conversation mode', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onChangeConfig = jest.fn();
    const Harness = () => {
      const [config, setConfig] = React.useState({
        code: 'knowledge_retrieval',
        parameters: [] as Metadata[],
      });
      return (
        <BuildinTool
          toolDefinition={{
            name: 'tool',
            description: 'desc',
            parameters: '{}',
          }}
          onChangeToolDefinition={jest.fn()}
          onChangeBuildinTool={jest.fn()}
          onChangeConfig={next => {
            onChangeConfig(next);
            setConfig(next);
          }}
          config={config}
        />
      );
    };

    render(<Harness />);

    await user.click(screen.getByRole('button', { name: 'Add rule' }));
    const addedRuleConfig = onChangeConfig.mock.calls.at(-1)?.[0];
    const addedCondition = addedRuleConfig.parameters.find(
      (m: Metadata) => m.getKey() === 'tool.condition',
    );
    const addedEntries = JSON.parse(addedCondition.getValue());
    expect(addedEntries).toHaveLength(2);

    await user.selectOptions(
      screen.getByTestId('tool-condition-key'),
      String('conversation_mode'),
    );
    await user.selectOptions(
      screen.getByTestId('tool-condition-source-value'),
      String('text'),
    );

    const lastConfig = onChangeConfig.mock.calls.at(-1)?.[0];
    const condition = lastConfig.parameters.find(
      (m: Metadata) => m.getKey() === 'tool.condition',
    );
    expect(condition.getValue()).toContain('"conversation_mode"');
    expect(condition.getValue()).toContain('"text"');
  });
});
