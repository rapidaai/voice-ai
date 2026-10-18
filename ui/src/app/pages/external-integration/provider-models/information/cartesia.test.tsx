import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { MemoryRouter } from 'react-router-dom';

import { CartesiaModelInformationPage } from './cartesia';

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: ({ title }: any) => <div data-testid="helmet">{title}</div>,
}));

jest.mock(
  '@/app/components/dialogs/provider/create-provider-credential-modal',
  () => ({
    CreateProviderCredentialDialog: () => (
      <div data-testid="create-provider-credential-dialog" />
    ),
  }),
);

jest.mock(
  '@/app/components/dialogs/provider/view-provider-credential-modal',
  () => ({
    ViewProviderCredentialDialog: () => (
      <div data-testid="view-provider-credential-dialog" />
    ),
  }),
);

jest.mock('@/app/components/layout/blocks/page-header-block', () => ({
  PageHeaderBlock: ({ children, className }: any) => (
    <header className={className}>{children}</header>
  ),
}));

jest.mock('@/app/components/layout/blocks/page-title-block', () => ({
  PageTitleBlock: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/layout/blocks/pagination-button-block', () => ({
  PaginationButtonBlock: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/layout/wrapper/blured-wrapper', () => ({
  BluredWrapper: ({ children }: any) => <div>{children}</div>,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  GhostButton: ({ children, renderIcon: Icon, ...props }: any) => (
    <button type="button" {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
  PrimaryButton: ({ children, renderIcon: Icon, ...props }: any) => (
    <button type="button" {...props}>
      {children}
      {Icon ? <Icon /> : null}
    </button>
  ),
}));

jest.mock(
  '@/app/pages/external-integration/provider-models/information/voice-catalog',
  () => ({
    VoiceCatalog: ({ actions, voices }: any) => (
      <section aria-label="Voice catalogue">
        <div data-testid="voice-catalog-actions">{actions}</div>
        {voices.map((voice: any) => (
          <div key={voice.voiceId}>{voice.title}</div>
        ))}
      </section>
    ),
  }),
);

jest.mock('@/hooks/use-model', () => ({
  useAllProviderCredentials: () => ({
    providerCredentials: [{ getProvider: () => 'cartesia' }],
  }),
}));

jest.mock('@/providers', () => ({
  CARTESIA_VOICE: () => [
    {
      id: 'voice-alpha',
      name: 'Alpha',
      description: 'First voice',
      language: 'English',
      mode: 'support',
    },
    {
      id: 'voice-beta',
      name: 'Beta',
      description: 'Second voice',
      language: 'Spanish',
      mode: 'sales',
    },
  ],
  TEXT_TO_SPEECH: () => ({
    code: 'cartesia',
    description: 'Cartesia provider',
    image: '/cartesia.svg',
    name: 'Cartesia',
  }),
}));

jest.mock('@carbon/icons-react', () => ({
  Add: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="add-icon"
    />
  ),
  ModelAlt: (props: any) => <svg {...props} />,
}));

const renderPage = (path = '/providers/cartesia') =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <CartesiaModelInformationPage />
    </MemoryRouter>,
  );

describe('CartesiaModelInformationPage', () => {
  it('renders provider status and add credential actions', async () => {
    renderPage();

    const providerHeader = screen.getByRole('banner');
    expect(providerHeader).toHaveClass('min-h-24');
    expect(providerHeader).not.toContainElement(
      screen.getByRole('button', { name: /add new credential/i }),
    );
    expect(screen.getByTestId('voice-catalog-actions')).toContainElement(
      screen.getByRole('button', { name: /add new credential/i }),
    );
    expect(screen.getByTestId('add-icon')).toBeInTheDocument();
    expect(screen.getByTestId('add-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );

    await waitFor(() =>
      expect(screen.getByText('Connected')).toBeInTheDocument(),
    );
  });

  it('maps provider voices while keeping page actions visible', () => {
    renderPage('/providers/cartesia');

    expect(screen.getByText('Beta')).toBeInTheDocument();
    expect(screen.getByText('Alpha')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /add new credential/i }));
  });

  it('shows a deterministic fallback when the provider logo fails', () => {
    renderPage();

    const logo = screen.getByRole('img', { name: 'Cartesia logo' });
    act(() => {
      logo.dispatchEvent(new Event('error'));
    });

    expect(screen.getByTestId('provider-logo-fallback')).toBeInTheDocument();
  });
});
