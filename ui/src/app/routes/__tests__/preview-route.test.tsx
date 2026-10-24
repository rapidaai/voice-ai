import React from 'react';
import '@testing-library/jest-dom';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { PreviewRoute } from '../preview';

jest.mock('@/app/components/layout/container/protected-box', () => ({
  ProtectedBox: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
}));

jest.mock('@/app/pages/preview-agent', () => ({
  PublicPreviewVoiceAgentPage: () => <div>Public preview</div>,
}));

const renderPreviewRoute = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/preview/*" element={<PreviewRoute />} />
        <Route
          path="/deployment/assistant/:assistantId/playground/chat"
          element={<div>Assistant chat debugger</div>}
        />
        <Route
          path="/deployment/assistant/:assistantId/playground/phone"
          element={<div>Assistant phone debugger</div>}
        />
      </Routes>
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
