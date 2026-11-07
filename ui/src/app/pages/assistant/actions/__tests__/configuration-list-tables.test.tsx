import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ConfigureAssistantAnalysisPage } from '@/app/pages/assistant/actions/configure-assistant-analysis';
import { ConfigureAssistantStoragePage } from '@/app/pages/assistant/actions/configure-assistant-storage';
import { ConfigureAssistantTelemetryPage } from '@/app/pages/assistant/actions/configure-assistant-telemetry';
import {
  useAssistantAnalysisPageStore,
  useAssistantStoragePageStore,
  useAssistantTelemetryPageStore,
} from '@/stores/assistant/actions';

const mockGetAnalysis = jest.fn();
const mockGetStorage = jest.fn();
const mockGetTelemetry = jest.fn();

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
    goToCreateAssistantAnalysis: jest.fn(),
    goToEditAssistantAnalysis: jest.fn(),
    goToCreateAssistantStorage: jest.fn(),
    goToEditAssistantStorage: jest.fn(),
    goToCreateAssistantTelemetry: jest.fn(),
    goToEditAssistantTelemetry: jest.fn(),
  }),
}));

jest.mock('@/app/pages/assistant/components/settings-shell-header', () => ({
  AssistantSettingsShellHeader: ({ title }: { title: string }) => (
    <header>{title}</header>
  ),
}));

jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-analysis/create-assistant-analysis',
  () => ({ CreateAssistantAnalysis: () => null }),
);
jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-analysis/update-assistant-analysis',
  () => ({ UpdateAssistantAnalysis: () => null }),
);
jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-storage/create-assistant-storage',
  () => ({ CreateAssistantStorage: () => null }),
);
jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-storage/update-assistant-storage',
  () => ({ UpdateAssistantStorage: () => null }),
);
jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-telemetry/create-assistant-telemetry',
  () => ({ CreateAssistantTelemetry: () => null }),
);
jest.mock(
  '@/app/pages/assistant/actions/configure-assistant-telemetry/update-assistant-telemetry',
  () => ({ UpdateAssistantTelemetry: () => null }),
);

jest.mock('react-hot-toast/headless', () => ({
  error: jest.fn(),
  success: jest.fn(),
}));

const option = (key: string, value: string) => ({
  getKey: () => key,
  getValue: () => value,
});

const analysis = {
  getId: () => 'analysis-1',
  getOptionsList: () => [
    option('name', 'Sentiment review'),
    option('endpoint_id', 'endpoint-1'),
    option('endpoint_version', 'v1'),
    option('execution_priority', '1'),
  ],
  getEnabled: () => true,
  getStatus: () => 'ACTIVE',
  getCreateddate: () => undefined,
};

const storage = {
  getId: () => 'storage-1',
  getProvider: () => 'aws-cloud',
  getConfigurationtype: () => 'storage',
  getOptionsList: () => [option('s3_bucket_name', 'recordings')],
  getEnabled: () => true,
  getStatus: () => 'ACTIVE',
  getCreateddate: () => undefined,
};

const telemetry = {
  getId: () => 'telemetry-1',
  getProvider: () => 'custom',
  getConfigurationtype: () => 'telemetry',
  getOptionsList: () => [option('endpoint', 'https://telemetry.example.test')],
  getEnabled: () => true,
  getStatus: () => 'ACTIVE',
  getCreateddate: () => undefined,
};

describe('assistant configuration list tables', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetAnalysis.mockImplementation(
      (_assistantId, _projectId, _token, _authId, _onError, onSuccess) =>
        onSuccess([analysis]),
    );
    mockGetStorage.mockImplementation(
      (_assistantId, _projectId, _token, _authId, _onError, onSuccess) =>
        onSuccess([storage]),
    );
    mockGetTelemetry.mockImplementation(
      (_assistantId, _projectId, _token, _authId, _onError, onSuccess) =>
        onSuccess([telemetry]),
    );

    useAssistantAnalysisPageStore.setState({
      analysises: [analysis] as any,
      page: 1,
      pageSize: 10,
      totalCount: 1,
      criteria: [],
      getAssistantAnalysis: mockGetAnalysis as any,
    });
    useAssistantStoragePageStore.setState({
      storages: [storage] as any,
      page: 1,
      pageSize: 10,
      totalCount: 1,
      criteria: [],
      getAssistantStorage: mockGetStorage as any,
    });
    useAssistantTelemetryPageStore.setState({
      telemetries: [telemetry] as any,
      page: 1,
      pageSize: 10,
      totalCount: 1,
      criteria: [],
      getAssistantTelemetry: mockGetTelemetry as any,
    });
  });

  it.each([
    ['Analysis', ConfigureAssistantAnalysisPage],
    ['Storage', ConfigureAssistantStoragePage],
    ['Telemetry', ConfigureAssistantTelemetryPage],
  ])('uses Carbon selection and bottom pagination for %s', async (_, Page) => {
    const interaction = userEvent.setup();
    const { container } = render(<Page />);

    await interaction.click(
      await screen.findByRole('radio', { name: 'Select row' }),
    );

    expect(screen.getByRole('button', { name: 'Edit' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Disable' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Delete' })).toBeInTheDocument();
    expect(container.querySelector('.cds--pagination')).toHaveClass('shrink-0');
  });

  it.each([
    ['Analysis', ConfigureAssistantAnalysisPage, mockGetAnalysis],
    ['Storage', ConfigureAssistantStoragePage, mockGetStorage],
    ['Telemetry', ConfigureAssistantTelemetryPage, mockGetTelemetry],
  ])('keeps an in-place table skeleton for %s', async (_, Page, fetchList) => {
    fetchList.mockImplementation(() => undefined);

    const { container } = render(<Page />);

    await waitFor(() => expect(fetchList).toHaveBeenCalled());
    const skeleton = container.querySelector(
      '.cds--data-table-container.cds--skeleton',
    );
    expect(skeleton).toBeInTheDocument();
    expect(skeleton?.parentElement).toHaveClass(
      'flex-1',
      'flex-col',
      'overflow-hidden',
    );
  });
});
