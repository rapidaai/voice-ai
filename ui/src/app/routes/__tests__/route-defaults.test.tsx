import React from 'react';
import '@testing-library/jest-dom';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { DeploymentRoute } from '../deployment';
import { OnbaordingRoute } from '../onborading';
import { StaticRoute } from '../static';

jest.mock('@/app/components/layout/container/protected-box', () => ({
  ProtectedBox: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

jest.mock('@/app/components/layout/container/mission-box', () => ({
  MissionBox: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@/app/pages/assistant', () => {
  const EmptyPage = () => null;
  return {
    DeploymentAssistantPage: () => <div>Assistant listing</div>,
    DeploymentCreateAssistantPage: EmptyPage,
    DeploymentViewAssistantPage: EmptyPage,
    DeploymentCreateVersionAssistantPage: EmptyPage,
    DeploymentConfigureAssistantWebDeploymentPage: EmptyPage,
    DeploymentEditAssistantWebDeploymentPage: EmptyPage,
    DeploymentConfigureAssistantCallDeploymentPage: EmptyPage,
    DeploymentEditAssistantCallDeploymentPage: EmptyPage,
    DeploymentConfigureAssistantToolPage: EmptyPage,
    DeploymentConfigureAssistantAnalysisPage: EmptyPage,
    DeploymentConfigureAssistantApiDeploymentPage: EmptyPage,
    DeploymentEditAssistantApiDeploymentPage: EmptyPage,
    DeploymentConfigureAssistantDebuggerDeploymentPage: EmptyPage,
    DeploymentEditAssistantDebuggerDeploymentPage: EmptyPage,
    DeploymentEditAssistantPage: EmptyPage,
    DeploymentConfigureAssistantWebhookPage: EmptyPage,
    DeploymentCreateAssistantWebhookPage: EmptyPage,
    DeploymentUpdateAssistantWebhookPage: EmptyPage,
    DeploymentConfigureAssistantTelemetryPage: EmptyPage,
    DeploymentCreateAssistantTelemetryPage: EmptyPage,
    DeploymentUpdateAssistantTelemetryPage: EmptyPage,
    DeploymentConfigureAssistantStoragePage: EmptyPage,
    DeploymentCreateAssistantStoragePage: EmptyPage,
    DeploymentUpdateAssistantStoragePage: EmptyPage,
    DeploymentConfigureAssistantAuthenticationPage: EmptyPage,
    DeploymentCreateAssistantAuthenticationPage: EmptyPage,
    DeploymentUpdateAssistantAuthenticationPage: EmptyPage,
    DeploymentConversationDetailPage: EmptyPage,
    DeploymentCreateAssistantAnalysisPage: EmptyPage,
    DeploymentUpdateAssistantAnalysisPage: EmptyPage,
    DeploymentUpdateAssistantToolPage: EmptyPage,
    DeploymentCreateAssistantToolPage: EmptyPage,
    DeploymentConfigureAssistantDeploymentPage: EmptyPage,
    DeploymentCreateWebsocketVersionAssistantPage: EmptyPage,
    DeploymentCreateAgentKitVersionAssistantPage: EmptyPage,
    DeploymentCreateAgentflowVersionAssistantPage: EmptyPage,
    DeploymentCreateAgentKitPage: EmptyPage,
    DeploymentCreateWebsocketPage: EmptyPage,
    DeploymentCreateAgentflowPage: EmptyPage,
  };
});

jest.mock('@/app/pages/assistant/view/assistant-view.layout', () => {
  const { Outlet } = jest.requireActual('react-router-dom');
  return { AssistantViewLayout: () => <Outlet /> };
});

jest.mock('@/app/pages/preview-agent', () => ({
  PreviewVoiceAgentV2Page: ({ embedded }: { embedded?: boolean }) => (
    <div>{embedded ? 'Embedded chat debugger' : 'Chat debugger'}</div>
  ),
  PreviewPhoneAgentPage: ({ embedded }: { embedded?: boolean }) => (
    <div>{embedded ? 'Embedded phone debugger' : 'Phone debugger'}</div>
  ),
}));

jest.mock('@/app/pages/endpoint', () => {
  const EmptyPage = () => null;
  return {
    DeploymentEndpointPage: EmptyPage,
    DeploymentCreateEndpointPage: EmptyPage,
    DeploymentViewEndpointPage: EmptyPage,
    DeploymentCreateVersionEndpointPage: EmptyPage,
    DeploymentConfigureEndpointPage: EmptyPage,
  };
});
jest.mock('@/app/pages/endpoint/view/endpoint-view.layout', () => {
  const { Outlet } = jest.requireActual('react-router-dom');
  return { EndpointViewLayout: () => <Outlet /> };
});

jest.mock('@/app/pages/user-onboarding', () => ({
  OnboardingCreateOrganizationPage: () => <div>Organization onboarding</div>,
  OnboardingCreateProjectPage: () => <div>Project onboarding</div>,
}));

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({
    theme: {
      brand: { name: 'Rapida AI' },
      links: {
        documentation: '/docs',
        privacy: '/privacy',
        support: '/support',
        terms: '/terms',
      },
    },
  }),
}));

jest.mock('@/app/components/layout/brand/branded-logo', () => ({
  BrandedLogo: () => <div>Rapida AI</div>,
}));

jest.mock('@/app/pages/static-pages', () => ({
  StaticPrivacyPage: () => <div>Privacy page</div>,
  StaticTermsPage: () => <div>Terms page</div>,
  StaticPageNotFoundPage: () => <div>Not found page</div>,
}));

const CurrentPath = () => {
  const location = useLocation();
  return <output>{location.pathname}</output>;
};

describe('route defaults', () => {
  it('redirects the deployment root to assistants', async () => {
    render(
      <MemoryRouter initialEntries={['/deployment']}>
        <CurrentPath />
        <Routes>
          <Route path="/deployment/*" element={<DeploymentRoute />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText('Assistant listing')).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByText('/deployment/assistant')).toBeInTheDocument(),
    );
  });

  it.each([
    [
      '/deployment/assistant/assistant-1/playground/chat',
      'Embedded chat debugger',
    ],
    [
      '/deployment/assistant/assistant-1/playground/phone',
      'Embedded phone debugger',
    ],
  ])('renders the native assistant playground at %s', (path, expectedText) => {
    render(
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/deployment/*" element={<DeploymentRoute />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText(expectedText)).toBeInTheDocument();
  });

  it('redirects the onboarding root to organization setup', async () => {
    render(
      <MemoryRouter initialEntries={['/onboarding']}>
        <CurrentPath />
        <Routes>
          <Route path="/onboarding/*" element={<OnbaordingRoute />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(
      await screen.findByText('Organization onboarding'),
    ).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.getByText('/onboarding/organization')).toBeInTheDocument(),
    );
  });

  it.each([
    ['/static/privacy-policy', 'Privacy page'],
    ['/static/terms-conditions', 'Terms page'],
    ['/static/missing', 'Not found page'],
  ])('renders the static subjourney for %s', (path, expectedText) => {
    render(
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/static/*" element={<StaticRoute />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText(expectedText)).toBeInTheDocument();
  });
});
