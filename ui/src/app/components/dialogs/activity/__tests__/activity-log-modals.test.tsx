import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import toast from 'react-hot-toast/headless';
import {
  ConnectionConfig,
  GetActivity,
  GetAssistantToolLog,
  GetHTTPLog,
  GetKnowledgeLog,
} from '@rapidaai/react';

import {
  ConversationLogContent,
  ConversationLogDialog,
  MessageMetadatas,
} from '../conversation-log-modal';
import { KnowledgeLogDialog } from '../knowledge-log-modal';
import { LLMLogContent, LLMLogDialog } from '../llm-log-modal';
import { ToolLogDialog } from '../tool-log-modal';
import { RequestLogDialog } from '../webhook-log-modal';
import { LogCodePanel } from '../log-modal-primitives';
import * as ActivityDialogs from '../index';

const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token-1', 'project-1'],
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/configs', () => ({
  connectionConfig: { api: 'test' },
}));

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
}));

jest.mock('@/utils/date', () => ({
  toHumanReadableDateTime: (value: unknown) => `date:${String(value)}`,
}));

jest.mock('@/app/components/dialogs/shared', () => ({
  RightSideModal: ({ children, label, modalOpen, title }: any) =>
    modalOpen ? (
      <aside aria-label={label} data-testid="right-side-modal">
        <h2>{title}</h2>
        {children}
      </aside>
    ) : null,
  OverviewRow: ({ children, label }: any) => (
    <div data-testid={`overview-${label}`}>{children}</div>
  ),
}));

jest.mock('@/app/components/ui/primitives', () => ({
  Tabs: ({
    children,
    onChange,
    selectedIndex,
    tabs,
    'aria-label': ariaLabel,
  }: any) => (
    <section aria-label={ariaLabel} data-selected-index={selectedIndex}>
      <div role="tablist">
        {tabs.map((tab: string, index: number) => (
          <button
            key={tab}
            type="button"
            role="tab"
            onClick={() => onChange(index)}
          >
            {tab}
          </button>
        ))}
      </div>
      {children}
    </section>
  ),
}));

jest.mock('@/app/components/ui/editor/code-highlighting', () => ({
  CodeHighlighting: ({ code, language }: any) => (
    <pre data-language={language}>{code}</pre>
  ),
}));

jest.mock('@/app/components/ui/editor/markdown-viewer', () => ({
  MarkdownViewer: ({ text }: any) => <article>{text}</article>,
}));

jest.mock('@/app/components/ui/feedback', () => ({
  CarbonStatusIndicator: ({ state }: any) => <span>{state}</span>,
  EmptyState: ({ subtitle, title }: any) => (
    <div role="status">
      <strong>{title}</strong>
      {subtitle}
    </div>
  ),
}));

jest.mock('@/app/components/domain/indicators/status', () => ({
  StatusIndicator: ({ state }: any) => <span>{state}</span>,
}));

jest.mock('@/app/components/domain/indicators/http-status', () => ({
  HttpStatusSpanIndicator: ({ status }: any) => <span>{status}</span>,
}));

jest.mock('@carbon/icons-react', () => ({
  Chat: () => <svg data-testid="chat-icon" />,
}));

jest.mock('@rapidaai/react', () => {
  class GetKnowledgeLogRequest {
    id = '';
    projectId = '';
    setId(value: string) {
      this.id = value;
    }
    setProjectid(value: string) {
      this.projectId = value;
    }
  }

  class GetAssistantToolLogRequest {
    id = '';
    projectId = '';
    setId(value: string) {
      this.id = value;
    }
    setProjectid(value: string) {
      this.projectId = value;
    }
  }

  class GetAssistantHTTPLogRequest {
    id = '';
    projectId = '';
    setId(value: string) {
      this.id = value;
    }
    setProjectid(value: string) {
      this.projectId = value;
    }
  }

  return {
    AssistantConversationMessage: class {},
    AssistantHTTPLog: class {},
    AssistantToolLog: class {},
    AuditLog: class {},
    KnowledgeLog: class {},
    Metadata: class {},
    ConnectionConfig: {
      WithDebugger: jest.fn((headers: unknown) => ({ headers })),
    },
    GetActivity: jest.fn(),
    GetAssistantHTTPLogRequest,
    GetAssistantToolLogRequest,
    GetAssistantToolLog: jest.fn(),
    GetHTTPLog: jest.fn(),
    GetKnowledgeLogRequest,
    GetKnowledgeLog: jest.fn(),
  };
});

const setModalOpen = jest.fn();
const fallbackError = 'Unable to resolve the request, please try again later.';

const structValue = (value: unknown) => ({
  toJavaScript: () => value,
});

const metricValue = (value: Record<string, unknown>) => ({
  toObject: () => value,
});

const metadataValue = (key: string, value: string) => ({
  getKey: () => key,
  getValue: () => value,
});

const serviceError = (message: string) => ({
  getHumanmessage: () => message,
});

const apiResponse = (
  success: boolean,
  data?: unknown,
  message = 'Activity log failed',
) => ({
  getSuccess: () => success,
  getData: () => data,
  getError: () => serviceError(message),
});

const conversationMessage = (
  body: string,
  metadata = [metadataValue('role', 'assistant')],
) =>
  ({
    getAssistantconversationid: () => 'conv-1',
    getBody: () => body,
    getMetricsList: () => [metricValue({ tokens: 42 })],
    getMetadataList: () => metadata,
  }) as any;

const knowledgeLog = () =>
  ({
    getStatus: () => 'SUCCESS',
    getTimetaken: () => '2400000',
    getCreateddate: () => '2026-09-19T10:00:00Z',
    getRequest: () => structValue({ question: 'Where is the policy?' }),
    getResponse: () => structValue({ answer: 'In the handbook.' }),
  }) as any;

const llmLog = () =>
  ({
    getStatus: () => 'SUCCESS',
    getTimetaken: () => 8200000,
    getCreateddate: () => '2026-09-19T11:00:00Z',
    getResponsestatus: () => 200,
    getExternalauditmetadatasList: () => [
      metadataValue('model_name', 'gpt-4.1'),
    ],
    getRequest: () => structValue({ prompt: 'Summarize' }),
    getResponse: () => structValue({ message: 'Done' }),
    getMetricsList: () => [metricValue({ inputTokens: 12 })],
  }) as any;

const toolLog = () =>
  ({
    getRequest: () => structValue({ tool: 'lookup' }),
    getResponse: () => structValue({ ok: true }),
  }) as any;

describe('activity log modal content', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders conversation message, metrics, metadata, and tab changes', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onTabChange = jest.fn();

    render(
      <ConversationLogContent
        currentAssistantMessage={conversationMessage('Hello from the trace')}
        selectedTab={0}
        onTabChange={onTabChange}
      />,
    );

    expect(screen.getByText('Hello from the trace')).toBeInTheDocument();
    expect(screen.getByText(/"tokens": 42/)).toHaveAttribute(
      'data-language',
      'json',
    );
    expect(screen.getByText('assistant')).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'Metrics' }));
    expect(onTabChange).toHaveBeenCalledWith(1);
  });

  it('renders empty states for missing conversation message and metadata', () => {
    render(
      <ConversationLogContent
        currentAssistantMessage={conversationMessage('', [])}
        selectedTab={0}
        onTabChange={jest.fn()}
      />,
    );

    expect(screen.getByText('No Message')).toBeInTheDocument();
    expect(screen.getByText('No Metadata')).toBeInTheDocument();
  });

  it('renders null JSON when a code panel has no value', () => {
    render(<LogCodePanel value={undefined} />);

    expect(screen.getByText('null')).toHaveAttribute('data-language', 'json');
  });

  it('falls back to null JSON when a value cannot be serialized', () => {
    render(<LogCodePanel value={() => undefined} />);

    expect(screen.getByText('null')).toHaveAttribute('data-language', 'json');
  });

  it('renders the metadata empty state directly', () => {
    render(<MessageMetadatas metadata={[]} />);

    expect(
      screen.getByText('There is no metadata for this message.'),
    ).toBeInTheDocument();
  });

  it('exposes the activity dialog barrel exports', () => {
    expect(ActivityDialogs.ConversationLogDialog).toBe(ConversationLogDialog);
    expect(ActivityDialogs.KnowledgeLogDialog).toBe(KnowledgeLogDialog);
    expect(ActivityDialogs.LLMLogDialog).toBe(LLMLogDialog);
    expect(ActivityDialogs.ToolLogDialog).toBe(ToolLogDialog);
    expect(ActivityDialogs.RequestLogDialog).toBe(RequestLogDialog);
  });

  it('omits the LLM response status row when no status was recorded', () => {
    render(
      <LLMLogContent
        activity={{ ...llmLog(), getResponsestatus: () => 0 }}
        additionalData={[]}
        selectedTab={0}
        onTabChange={jest.fn()}
      />,
    );

    expect(
      screen.queryByTestId('overview-Response Status'),
    ).not.toBeInTheDocument();
  });
});

describe('activity log modal containers', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('loads knowledge log details and sends debugger credentials', async () => {
    (GetKnowledgeLog as jest.Mock).mockResolvedValue(
      apiResponse(true, knowledgeLog()),
    );

    render(
      <KnowledgeLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="knowledge-log-1"
      />,
    );

    await waitFor(() =>
      expect(screen.getByText('SUCCESS')).toBeInTheDocument(),
    );

    const request = (GetKnowledgeLog as jest.Mock).mock.calls[0][1];
    expect(request.id).toBe('knowledge-log-1');
    expect(request.projectId).toBe('project-1');
    expect(ConnectionConfig.WithDebugger).toHaveBeenCalledWith({
      authorization: 'token-1',
      projectId: 'project-1',
      userId: 'user-1',
    });
    expect(mockShowLoader).toHaveBeenCalledWith('overlay');
    expect(mockHideLoader).toHaveBeenCalledTimes(1);
    expect(screen.getByText('2.4ms')).toBeInTheDocument();
    expect(screen.getByText(/Where is the policy/)).toBeInTheDocument();
  });

  it('reports knowledge log API errors', async () => {
    (GetKnowledgeLog as jest.Mock).mockResolvedValue(
      apiResponse(false, undefined, 'Knowledge log missing'),
    );

    render(
      <KnowledgeLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="knowledge-log-2"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith('Knowledge log missing'),
    );
    expect(toast.error).toHaveBeenCalledWith(fallbackError);
    expect(mockHideLoader).toHaveBeenCalledTimes(1);
  });

  it('reports rejected knowledge log requests', async () => {
    (GetKnowledgeLog as jest.Mock).mockRejectedValue(new Error('network'));

    render(
      <KnowledgeLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="knowledge-log-3"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(fallbackError),
    );
    expect(mockHideLoader).toHaveBeenCalledTimes(1);
  });

  it('handles successful knowledge log responses without data', async () => {
    (GetKnowledgeLog as jest.Mock).mockResolvedValue(apiResponse(true));

    render(
      <KnowledgeLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="knowledge-log-4"
      />,
    );

    await waitFor(() => expect(mockHideLoader).toHaveBeenCalledTimes(1));
    expect(toast.error).not.toHaveBeenCalled();
    expect(screen.queryByText('SUCCESS')).not.toBeInTheDocument();
  });

  it('reports the knowledge log fallback without a server message', async () => {
    (GetKnowledgeLog as jest.Mock).mockResolvedValue({
      getSuccess: () => false,
      getError: () => undefined,
    });

    render(
      <KnowledgeLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="knowledge-log-5"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(fallbackError),
    );
    expect(toast.error).toHaveBeenCalledTimes(1);
  });

  it('loads LLM log details from the callback client', async () => {
    (GetActivity as jest.Mock).mockImplementation(
      (_config, _projectId, _activityId, callback) => {
        callback(null, apiResponse(true, llmLog()));
      },
    );

    render(
      <LLMLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="llm-log-1"
      />,
    );

    await waitFor(() =>
      expect(screen.getByText('gpt-4.1')).toBeInTheDocument(),
    );
    expect(screen.getByText('8.2ms')).toBeInTheDocument();
    expect(screen.getByText('200')).toBeInTheDocument();
    expect(screen.getByText(/"inputTokens": 12/)).toBeInTheDocument();
  });

  it('reports LLM callback errors', async () => {
    (GetActivity as jest.Mock).mockImplementation(
      (_config, _projectId, _activityId, callback) => {
        callback(serviceError('LLM request failed'), null);
      },
    );

    render(
      <LLMLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="llm-log-2"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith('LLM request failed'),
    );
    expect(toast.error).toHaveBeenCalledWith(fallbackError);
  });

  it('reports the LLM fallback when no human message is available', async () => {
    (GetActivity as jest.Mock).mockImplementation(
      (_config, _projectId, _activityId, callback) => {
        callback(null, {
          getSuccess: () => false,
          getError: () => undefined,
        });
      },
    );

    render(
      <LLMLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="llm-log-3"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(fallbackError),
    );
    expect(toast.error).toHaveBeenCalledTimes(1);
  });

  it('handles successful LLM responses without data', async () => {
    (GetActivity as jest.Mock).mockImplementation(
      (_config, _projectId, _activityId, callback) => {
        callback(null, apiResponse(true));
      },
    );

    render(
      <LLMLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="llm-log-4"
      />,
    );

    await waitFor(() => expect(mockHideLoader).toHaveBeenCalledTimes(1));
    expect(toast.error).not.toHaveBeenCalled();
    expect(screen.queryByText('SUCCESS')).not.toBeInTheDocument();
  });

  it('loads tool log details', async () => {
    (GetAssistantToolLog as jest.Mock).mockResolvedValue(
      apiResponse(true, toolLog()),
    );

    render(
      <ToolLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="tool-log-1"
      />,
    );

    await waitFor(() =>
      expect(screen.getByText(/"tool": "lookup"/)).toBeInTheDocument(),
    );
    expect(screen.getByText(/"ok": true/)).toBeInTheDocument();
  });

  it('reports tool log API errors', async () => {
    (GetAssistantToolLog as jest.Mock).mockResolvedValue(
      apiResponse(false, undefined, 'Tool log missing'),
    );

    render(
      <ToolLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="tool-log-3"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith('Tool log missing'),
    );
    expect(toast.error).toHaveBeenCalledWith(fallbackError);
  });

  it('reports tool log request failures', async () => {
    (GetAssistantToolLog as jest.Mock).mockRejectedValue(new Error('network'));

    render(
      <ToolLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="tool-log-2"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(fallbackError),
    );
    expect(mockHideLoader).toHaveBeenCalledTimes(1);
  });

  it('handles successful tool log responses without data', async () => {
    (GetAssistantToolLog as jest.Mock).mockResolvedValue(apiResponse(true));

    render(
      <ToolLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="tool-log-4"
      />,
    );

    await waitFor(() => expect(mockHideLoader).toHaveBeenCalledTimes(1));
    expect(toast.error).not.toHaveBeenCalled();
  });

  it('reports the tool log fallback without a server message', async () => {
    (GetAssistantToolLog as jest.Mock).mockResolvedValue({
      getSuccess: () => false,
      getError: () => undefined,
    });

    render(
      <ToolLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentActivityId="tool-log-5"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(fallbackError),
    );
    expect(toast.error).toHaveBeenCalledTimes(1);
  });

  it('loads webhook request log details', async () => {
    (GetHTTPLog as jest.Mock).mockResolvedValue(apiResponse(true, toolLog()));

    render(
      <RequestLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentRequestLogId="request-log-1"
      />,
    );

    await waitFor(() =>
      expect(screen.getByText(/"tool": "lookup"/)).toBeInTheDocument(),
    );

    const request = (GetHTTPLog as jest.Mock).mock.calls[0][1];
    expect(request.id).toBe('request-log-1');
    expect(request.projectId).toBe('project-1');
    expect((GetHTTPLog as jest.Mock).mock.calls[0][2]).toEqual({
      authorization: 'token-1',
      'x-auth-id': 'user-1',
      'x-project-id': 'project-1',
    });
  });

  it('reports webhook request log API errors', async () => {
    (GetHTTPLog as jest.Mock).mockResolvedValue(
      apiResponse(false, undefined, 'Request log missing'),
    );

    render(
      <RequestLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentRequestLogId="request-log-2"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith('Request log missing'),
    );
    expect(toast.error).toHaveBeenCalledWith(fallbackError);
  });

  it('reports rejected webhook request log calls', async () => {
    (GetHTTPLog as jest.Mock).mockRejectedValue(new Error('network'));

    render(
      <RequestLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentRequestLogId="request-log-3"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(fallbackError),
    );
    expect(mockHideLoader).toHaveBeenCalledTimes(1);
  });

  it('handles successful webhook request log responses without data', async () => {
    (GetHTTPLog as jest.Mock).mockResolvedValue(apiResponse(true));

    render(
      <RequestLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentRequestLogId="request-log-4"
      />,
    );

    await waitFor(() => expect(mockHideLoader).toHaveBeenCalledTimes(1));
    expect(toast.error).not.toHaveBeenCalled();
  });

  it('reports the webhook request log fallback without a server message', async () => {
    (GetHTTPLog as jest.Mock).mockResolvedValue({
      getSuccess: () => false,
      getError: () => undefined,
    });

    render(
      <RequestLogDialog
        modalOpen
        setModalOpen={setModalOpen}
        currentRequestLogId="request-log-5"
      />,
    );

    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(fallbackError),
    );
    expect(toast.error).toHaveBeenCalledTimes(1);
  });

  it('does not render closed conversation drawers', () => {
    render(
      <ConversationLogDialog
        modalOpen={false}
        setModalOpen={setModalOpen}
        currentAssistantMessage={conversationMessage('Hidden')}
      />,
    );

    expect(screen.queryByText('Hidden')).not.toBeInTheDocument();
  });
});
