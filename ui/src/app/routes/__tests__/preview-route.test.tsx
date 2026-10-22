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
  PreviewPhoneAgentPage: () => <div>Phone preview</div>,
  PreviewVoiceAgentPage: () => <div>Classic debugger</div>,
  PreviewVoiceAgentV2Page: () => <div>Debugger v2</div>,
  PublicPreviewVoiceAgentPage: () => <div>Public preview</div>,
}));

const renderPreviewRoute = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/preview/*" element={<PreviewRoute />} />
      </Routes>
    </MemoryRouter>,
  );

describe('PreviewRoute', () => {
  it('renders the coded debugger design on the v2 URL', () => {
    renderPreviewRoute('/preview/v2/chat/assistant-123');

    expect(screen.getByText('Debugger v2')).toBeInTheDocument();
  });

  it('keeps the existing debugger URL on the classic page', () => {
    renderPreviewRoute('/preview/chat/assistant-123');

    expect(screen.getByText('Classic debugger')).toBeInTheDocument();
    expect(screen.queryByText('Debugger v2')).not.toBeInTheDocument();
  });
});
