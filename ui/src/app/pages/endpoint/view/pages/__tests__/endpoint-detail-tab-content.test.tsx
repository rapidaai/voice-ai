import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { EndpointDetailTabContent } from '@/app/pages/endpoint/view/pages';

jest.mock('@/app/pages/endpoint/view/pages/overview-page', () => ({
  EndpointOverviewPage: ({
    currentEndpoint,
    currentEndpointProviderModel,
  }: any) => (
    <section>
      Overview summary {currentEndpoint.getName()}{' '}
      {currentEndpointProviderModel.getId()}
    </section>
  ),
}));

jest.mock('@/app/pages/endpoint/view/try-playground', () => ({
  Playground: ({ currentEndpoint, currentEndpointProviderModel }: any) => (
    <section>
      Playground page {currentEndpoint.getName()}{' '}
      {currentEndpointProviderModel.getId()}
    </section>
  ),
}));

jest.mock('@/app/pages/endpoint/view/traces', () => ({
  EndpointTraces: ({ currentEndpoint }: any) => (
    <section>Logs page {currentEndpoint.getName()}</section>
  ),
}));

jest.mock('@/app/pages/endpoint/view/version-list', () => ({
  Version: ({ currentEndpoint, onReload }: any) => (
    <section>
      Versions page {currentEndpoint.getName()}
      <button type="button" onClick={onReload}>
        Reload versions
      </button>
    </section>
  ),
}));

jest.mock('@/app/pages/endpoint/view/pages/settings-page', () => ({
  EndpointSettingsPage: ({ currentEndpoint }: any) => (
    <section>Settings page {currentEndpoint.getName()}</section>
  ),
}));

const makeEndpoint = () =>
  ({
    getId: () => 'endpoint-1',
    getName: () => 'Production endpoint',
  }) as any;

const makeEndpointProviderModel = () =>
  ({
    getId: () => 'epm-1',
  }) as any;

describe('EndpointDetailTabContent', () => {
  it('renders the overview page for the overview tab', () => {
    render(
      <EndpointDetailTabContent
        activeTab="overview"
        currentEndpoint={makeEndpoint()}
        currentEndpointProviderModel={makeEndpointProviderModel()}
        onReload={jest.fn()}
      />,
    );

    expect(
      screen.getByText('Overview summary Production endpoint epm-1'),
    ).toBeInTheDocument();
  });

  it('renders the playground only for the playground tab', () => {
    render(
      <EndpointDetailTabContent
        activeTab="playground"
        currentEndpoint={makeEndpoint()}
        currentEndpointProviderModel={makeEndpointProviderModel()}
        onReload={jest.fn()}
      />,
    );

    expect(
      screen.getByText('Playground page Production endpoint epm-1'),
    ).toBeInTheDocument();
  });

  it('renders logs and versions pages from their route keys', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onReload = jest.fn();
    const { rerender } = render(
      <EndpointDetailTabContent
        activeTab="logs"
        currentEndpoint={makeEndpoint()}
        currentEndpointProviderModel={makeEndpointProviderModel()}
        onReload={onReload}
      />,
    );

    expect(
      screen.getByText('Logs page Production endpoint'),
    ).toBeInTheDocument();

    rerender(
      <EndpointDetailTabContent
        activeTab="versions"
        currentEndpoint={makeEndpoint()}
        currentEndpointProviderModel={makeEndpointProviderModel()}
        onReload={onReload}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Reload versions' }));
    expect(
      screen.getByText(/Versions page Production endpoint/),
    ).toBeInTheDocument();
    expect(onReload).toHaveBeenCalledTimes(1);
  });

  it('renders endpoint settings from the settings route key', () => {
    render(
      <EndpointDetailTabContent
        activeTab="settings"
        currentEndpoint={makeEndpoint()}
        currentEndpointProviderModel={makeEndpointProviderModel()}
        onReload={jest.fn()}
      />,
    );

    expect(
      screen.getByText('Settings page Production endpoint'),
    ).toBeInTheDocument();
  });

  it('renders nothing until endpoint data is ready', () => {
    const { container } = render(
      <EndpointDetailTabContent
        activeTab="overview"
        currentEndpoint={null}
        currentEndpointProviderModel={makeEndpointProviderModel()}
        onReload={jest.fn()}
      />,
    );

    expect(container).toBeEmptyDOMElement();
  });

  it('falls back to overview for unknown endpoint tab paths', () => {
    render(
      <EndpointDetailTabContent
        activeTab="unknown-tab"
        currentEndpoint={makeEndpoint()}
        currentEndpointProviderModel={makeEndpointProviderModel()}
        onReload={jest.fn()}
      />,
    );

    expect(
      screen.getByText('Overview summary Production endpoint epm-1'),
    ).toBeInTheDocument();
  });
});
