import { renderHook } from '@testing-library/react';

import { useGlobalNavigation } from '@/hooks/use-global-navigator';

const mockNavigate = jest.fn();

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
}));

describe('useGlobalNavigation', () => {
  beforeEach(() => {
    mockNavigate.mockReset();
  });

  it('navigates backward and to direct routes', () => {
    const { result } = renderHook(() => useGlobalNavigation());

    result.current.goBack();
    result.current.goTo('/custom/path');
    result.current.goToDashboard();

    expect(mockNavigate).toHaveBeenNthCalledWith(1, -1);
    expect(mockNavigate).toHaveBeenNthCalledWith(2, '/custom/path');
    expect(mockNavigate).toHaveBeenNthCalledWith(3, '/dashboard');
  });

  it('builds assistant and deployment routes correctly', () => {
    const { result } = renderHook(() => useGlobalNavigation());

    result.current.goToAssistant('a-1');
    result.current.goToAssistantVersions('a-1');
    result.current.goToCreateAssistantVersion('a-1');
    result.current.goToConfigureApi('a-1');
    result.current.goToEditApi('a-1', 'api-1');
    result.current.goToConfigureCall('a-1');
    result.current.goToEditCall('a-1', 'dep-1');
    result.current.goToEditWeb('a-1', 'web-1');
    result.current.goToEditDebugger('a-1', 'dbg-1');
    result.current.goToConfigureDebuggerExperience('a-1');
    result.current.goToConfigureDebuggerSTT('a-1');
    result.current.goToConfigureDebuggerTTS('a-1');
    result.current.goToCreateAssistantTool('a-1');

    expect(mockNavigate).toHaveBeenNthCalledWith(
      1,
      '/deployment/assistant/a-1',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      2,
      '/deployment/assistant/a-1/version-history',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      3,
      '/deployment/assistant/a-1/create-new-version',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      4,
      '/deployment/assistant/a-1/deployment/api',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      5,
      '/deployment/assistant/a-1/deployment/api/api-1',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      6,
      '/deployment/assistant/a-1/deployment/call',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      7,
      '/deployment/assistant/a-1/deployment/call/dep-1',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      8,
      '/deployment/assistant/a-1/deployment/web/web-1',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      9,
      '/deployment/assistant/a-1/deployment/debugger/dbg-1',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      10,
      '/deployment/assistant/a-1/deployment/debugger?editMode=section&section=experience',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      11,
      '/deployment/assistant/a-1/deployment/debugger?editMode=section&section=stt',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      12,
      '/deployment/assistant/a-1/deployment/debugger?editMode=section&section=tts',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      13,
      '/deployment/assistant/a-1/configure-tool/create',
    );
  });

  it('builds knowledge and integration routes correctly', () => {
    const { result } = renderHook(() => useGlobalNavigation());

    result.current.goToKnowledge('k-1');
    result.current.goToKnowledgeAddManualFile('k-1');
    result.current.goToKnowledgeAddCloudFile('k-1');
    result.current.goToKnowledgeAddStructureFile('k-1');
    result.current.goToModelInformation('openai');

    expect(mockNavigate).toHaveBeenNthCalledWith(1, '/knowledge/k-1');
    expect(mockNavigate).toHaveBeenNthCalledWith(
      2,
      '/knowledge/k-1/add-knowledge-file',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      3,
      '/knowledge/k-1/add-cloud-file',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      4,
      '/knowledge/k-1/add-structure-file',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      5,
      '/integration/models/openai',
    );
  });

  it('opens preview windows for chat and call routes', () => {
    const openSpy = jest
      .spyOn(window, 'open')
      .mockImplementation(() => null as unknown as Window);

    const { result } = renderHook(() => useGlobalNavigation());

    result.current.goToAssistantPreview('assistant-1');
    result.current.goToAssistantPreviewCall('assistant-1');

    expect(openSpy).toHaveBeenNthCalledWith(1, '/preview/chat/assistant-1');
    expect(openSpy).toHaveBeenNthCalledWith(2, '/preview/call/assistant-1');

    openSpy.mockRestore();
  });

  it('builds telemetry routes with visible query filters', () => {
    const { result } = renderHook(() => useGlobalNavigation());

    result.current.goToConversationTelemetry('2340105440068632576');
    result.current.goToMessageTelemetry(
      'assistant-a49f2845-68ec-4a59-a30d-5e2b30df87bf',
    );

    expect(mockNavigate).toHaveBeenNthCalledWith(
      1,
      '/logs/traces?query=conversation%3A2340105440068632576',
    );
    expect(mockNavigate).toHaveBeenNthCalledWith(
      2,
      '/logs/traces?query=message%3Aa49f2845-68ec-4a59-a30d-5e2b30df87bf',
    );
  });
});
