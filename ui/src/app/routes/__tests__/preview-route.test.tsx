import React from 'react';
import '@testing-library/jest-dom';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, useRoutes } from 'react-router-dom';
import { previewRoute } from '../preview';

jest.mock('@/app/components/layout/container/protected-box', () => ({
  ProtectedBox: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

jest.mock('@/app/pages/preview-agent', () => ({
  PublicPreviewVoiceAgentPage: () => <div>Public preview</div>,
}));

const PreviewRoutes = () =>
  useRoutes([
    previewRoute,
    {
      path: '/deployment/assistant/:assistantId/playground/chat',
      element: <div>Assistant chat debugger</div>,
    },
    {
      path: '/deployment/assistant/:assistantId/playground/phone',
      element: <div>Assistant phone debugger</div>,
    },
  ]);

const renderPreviewRoute = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <PreviewRoutes />
    </MemoryRouter>,
  );

describe('PreviewRoute', () => {
  it.each([
    ['/preview/chat/assistant-123', 'Assistant chat debugger'],
    ['/preview/v2/chat/assistant-123', 'Assistant chat debugger'],
    ['/preview/call/assistant-123', 'Assistant phone debugger'],
  ])('redirects %s to the assistant playground', (path, destination) => {
    renderPreviewRoute(path);

    expect(screen.getByText(destination)).toBeInTheDocument();
  });

  it('preserves the public assistant preview', () => {
    renderPreviewRoute('/preview/public/assistant/assistant-123');

    expect(screen.getByText('Public preview')).toBeInTheDocument();
  });
});
