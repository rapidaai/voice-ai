import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfigureAssistantToolPage } from './index';

const mockGetAssistantTool = jest.fn();
const mockDeleteAssistantTool = jest.fn();
const mockGoToEditAssistantTool = jest.fn();

const tool = {
  getId: () => 'tool-1',
  getName: () => 'Order lookup',
  getExecutionmethod: () => 'mcp',
  getExecutionoptionsList: () => [],
  getDescription: () => 'Looks up an order by reference.',
  getCreateddate: () => undefined,
};

const mockStore = {
  tools: [tool],
  getAssistantTool: mockGetAssistantTool,
  deleteAssistantTool: mockDeleteAssistantTool,
};

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ assistantId: 'assistant-1' }),
}));

jest.mock('@/hooks/use-credential', () => ({
  useCurrentCredential: () => ({
    authId: 'user-1',
    token: 'token-1',
    projectId: 'project-1',
  }),
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToCreateAssistantTool: jest.fn(),
    goToEditAssistantTool: mockGoToEditAssistantTool,
  }),
}));

jest.mock('@/stores/assistant/actions', () => ({
  useAssistantToolPageStore: () => mockStore,
}));

jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-tool/create-assistant-tool',
  () => ({ CreateTool: () => null }),
);

jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-tool/update-assistant-tool',
  () => ({ UpdateTool: () => null }),
);

jest.mock('react-hot-toast/headless', () => ({
  success: jest.fn(),
  error: jest.fn(),
}));

describe('ConfigureAssistantToolPage', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockStore.tools = [tool];
    mockGetAssistantTool.mockImplementation(
      (_assistantId, _projectId, _token, _authId, _onError, onSuccess) =>
        onSuccess([tool]),
    );
  });

  it('uses Carbon radio selection and batch actions for tool actions', async () => {
    const user = userEvent.setup();
    const { container } = render(<ConfigureAssistantToolPage />);

    await screen.findByText('Order lookup');

    const selection = screen.getByRole('radio', { name: 'Select row' });
    expect(selection.closest('td')).toHaveClass('cds--table-column-radio');
    expect(container.querySelector('.cds--batch-actions')).toBeNull();

    await user.click(selection);

    const batchActions = container.querySelector('.cds--batch-actions');
    expect(batchActions).toHaveAttribute('aria-hidden', 'false');
    expect(batchActions).toHaveTextContent('1 item selected');

    await user.click(
      within(batchActions as HTMLElement).getByRole('button', {
        name: 'Edit',
      }),
    );

    expect(mockGoToEditAssistantTool).toHaveBeenCalledWith(
      'assistant-1',
      'tool-1',
    );
  });

  it('asks for confirmation before deleting the selected tool', async () => {
    const user = userEvent.setup();
    const { container } = render(<ConfigureAssistantToolPage />);

    await user.click(await screen.findByRole('radio', { name: 'Select row' }));
    await user.click(
      within(
        container.querySelector('.cds--batch-actions') as HTMLElement,
      ).getByRole('button', { name: 'Delete' }),
    );

    expect(mockDeleteAssistantTool).not.toHaveBeenCalled();
    expect(screen.getByRole('dialog')).toHaveTextContent('Are you sure?');

    await user.click(
      within(screen.getByRole('dialog')).getByRole('button', {
        name: 'Confirm',
      }),
    );

    expect(mockDeleteAssistantTool).toHaveBeenCalledWith(
      'assistant-1',
      'tool-1',
      'project-1',
      'token-1',
      'user-1',
      expect.any(Function),
      expect.any(Function),
    );
  });

  it('keeps an in-place Carbon table skeleton while tools load', async () => {
    mockGetAssistantTool.mockImplementation(() => undefined);

    const { container } = render(<ConfigureAssistantToolPage />);

    await waitFor(() => expect(mockGetAssistantTool).toHaveBeenCalled());
    expect(
      screen.getByRole('heading', { name: 'Tool & MCP' }),
    ).toBeInTheDocument();

    const skeleton = container.querySelector(
      '.cds--data-table-container.cds--skeleton',
    );
    expect(skeleton).toBeInTheDocument();
    expect(skeleton?.parentElement).toHaveClass(
      'flex-1',
      'flex-col',
      'overflow-hidden',
    );
    expect(skeleton?.parentElement).not.toHaveClass(
      'items-center',
      'justify-center',
    );
  });
});
