import React from 'react';
import { act, render, within } from '@testing-library/react';
import '@testing-library/jest-dom';

import { AssistantPage } from '@/app/pages/assistant/listing';
import { EndpointPage } from '@/app/pages/endpoint/listing';

const mockNavigate = jest.fn();
const mockShowLoader = jest.fn();
const mockHideLoader = jest.fn();
const mockSearchParams = new URLSearchParams();

const mockAssistantState = {
  assistants: [],
  totalCount: 0,
  page: 1,
  pageSize: 10,
  criteria: [],
  setCriterias: jest.fn(),
  setPage: jest.fn(),
  setPageSize: jest.fn(),
  onGetAllAssistant: jest.fn(),
};

const mockEndpointState = {
  endpoints: [],
  totalCount: 0,
  page: 1,
  pageSize: 10,
  criteria: [],
  columns: [],
  setCriterias: jest.fn(),
  setPage: jest.fn(),
  setPageSize: jest.fn(),
  onGetAllEndpoint: jest.fn(),
};

let pageHeaderIntersectionCallback: IntersectionObserverCallback;

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => mockNavigate,
  useSearchParams: () => [mockSearchParams],
}));

jest.mock('@/hooks/use-credential', () => ({
  useCredential: () => ['user-1', 'token', 'project-1'],
}));

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: () => null,
}));

jest.mock('@/stores/app', () => ({
  useRapidaStore: () => ({
    loading: false,
    showLoader: mockShowLoader,
    hideLoader: mockHideLoader,
  }),
}));

jest.mock('@/stores/assistant/assistant.store', () => ({
  useAssistantPageStore: () => mockAssistantState,
}));

jest.mock('@/stores/endpoint', () => ({
  useEndpointPageStore: () => mockEndpointState,
}));

jest.mock('@/app/pages/assistant/listing/assistant-query-search', () => ({
  AssistantQuerySearch: () => <input aria-label="Search assistants" />,
  getAssistantSearchCriteria: () => [],
}));

jest.mock('@/app/pages/endpoint/listing/endpoint-query-search', () => ({
  EndpointQuerySearch: () => <input aria-label="Search endpoints" />,
  getEndpointSearchCriteria: () => [],
}));

describe('deployment listing page headers', () => {
  beforeAll(() => {
    window.IntersectionObserver = class IntersectionObserver {
      readonly root = null;
      readonly rootMargin = '';
      readonly thresholds = [];

      constructor(callback: IntersectionObserverCallback) {
        pageHeaderIntersectionCallback = callback;
      }

      observe() {}
      unobserve() {}
      disconnect() {}
      takeRecords() {
        return [];
      }
    };
  });

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('places the assistant context and create action in the reusable page header', () => {
    const { container } = render(<AssistantPage />);

    const pageHeader = container.querySelector('[data-page-header]');
    expect(pageHeader).not.toBeNull();
    expect(
      within(pageHeader as HTMLElement).getByRole('heading', {
        name: 'Assistants',
      }),
    ).toBeInTheDocument();
    expect(
      within(pageHeader as HTMLElement).getByText(
        'Create and manage assistants, versions, deployments, and connected experiences across your workspace.',
      ),
    ).toBeInTheDocument();
    expect(within(pageHeader as HTMLElement).getByText('0 of 0')).toBeVisible();
    expect(
      within(pageHeader as HTMLElement).getByRole('button', {
        name: 'Create new assistant',
      }),
    ).toBeVisible();

    const toolbar = container.querySelector('.cds--table-toolbar');
    expect(toolbar?.closest('[data-page-header-sticky-toolbar]')).toHaveClass(
      'sticky',
      'top-10',
      'z-10',
    );
    expect(
      within(toolbar as HTMLElement).getByRole('button', { name: 'Refresh' }),
    ).toBeVisible();
    expect(
      within(toolbar as HTMLElement).queryByRole('button', {
        name: 'Create new assistant',
      }),
    ).not.toBeInTheDocument();

    act(() => {
      pageHeaderIntersectionCallback(
        [{ isIntersecting: false } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      );
    });

    const stickyHeader = container.querySelector('[data-page-header-sticky]');
    expect(stickyHeader).toHaveAttribute('data-stuck', 'true');
    expect(
      within(stickyHeader as HTMLElement).getByRole('button', {
        name: 'Create new assistant',
      }),
    ).toHaveClass('cds--btn--md');
  });

  it('places the endpoint context and create action in the reusable page header', () => {
    const { container } = render(<EndpointPage />);

    const pageHeader = container.querySelector('[data-page-header]');
    expect(pageHeader).not.toBeNull();
    expect(
      within(pageHeader as HTMLElement).getByRole('heading', {
        name: 'Hosted Endpoints',
      }),
    ).toBeInTheDocument();
    expect(
      within(pageHeader as HTMLElement).getByText(
        'Create and manage hosted model endpoints, versions, and runtime performance across your workspace.',
      ),
    ).toBeInTheDocument();
    expect(within(pageHeader as HTMLElement).getByText('0 of 0')).toBeVisible();
    expect(
      within(pageHeader as HTMLElement).getByRole('button', {
        name: 'Create new endpoint',
      }),
    ).toBeVisible();

    const toolbar = container.querySelector('.cds--table-toolbar');
    expect(toolbar?.closest('[data-page-header-sticky-toolbar]')).toHaveClass(
      'sticky',
      'top-10',
      'z-10',
    );
    expect(
      within(toolbar as HTMLElement).getByRole('button', { name: 'Refresh' }),
    ).toBeVisible();
    expect(
      within(toolbar as HTMLElement).queryByRole('button', {
        name: 'Create new endpoint',
      }),
    ).not.toBeInTheDocument();

    act(() => {
      pageHeaderIntersectionCallback(
        [{ isIntersecting: false } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      );
    });

    const stickyHeader = container.querySelector('[data-page-header-sticky]');
    expect(stickyHeader).toHaveAttribute('data-stuck', 'true');
    expect(
      within(stickyHeader as HTMLElement).getByRole('button', {
        name: 'Create new endpoint',
      }),
    ).toHaveClass('cds--btn--md');
  });
});
