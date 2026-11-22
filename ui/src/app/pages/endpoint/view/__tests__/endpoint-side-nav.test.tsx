import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { EndpointSideNav } from '../endpoint-side-nav';

describe('EndpointSideNav', () => {
  beforeEach(() => {
    window.localStorage.clear();
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });
  });

  it('matches the global sidebar surface and collapse control', () => {
    render(
      <MemoryRouter
        initialEntries={['/deployment/endpoint/endpoint-1/overview']}
      >
        <EndpointSideNav endpointId="endpoint-1" />
      </MemoryRouter>,
    );

    const navigation = screen.getByLabelText('Endpoint actions');
    expect(navigation.parentElement).toHaveClass(
      'bg-shell',
      'text-foreground',
      'w-64',
    );
    expect(navigation).toHaveClass('bg-transparent!', 'transform-none!');

    const toggle = screen.getByRole('button', {
      name: 'Collapse endpoint navigation',
    });
    expect(toggle).toHaveClass('cds--btn--ghost');
    expect(toggle.querySelector('svg')).toBeInTheDocument();
    expect(toggle).toHaveTextContent('Collapse');
  });

  it('keeps navigation and toggle icons visible in the collapsed rail', async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={['/deployment/endpoint/endpoint-1/logs']}>
        <EndpointSideNav endpointId="endpoint-1" />
      </MemoryRouter>,
    );

    await user.click(
      screen.getByRole('button', {
        name: 'Collapse endpoint navigation',
      }),
    );

    const navigation = screen.getByLabelText('Endpoint actions');
    expect(navigation.parentElement).toHaveClass('w-12');
    expect(navigation).toHaveClass('transform-none!');
    expect(navigation).not.toHaveClass('translate-x-0!');
    expect(
      navigation.querySelector('.cds--side-nav__icon svg'),
    ).toBeInTheDocument();

    const toggle = screen.getByRole('button', {
      name: 'Expand endpoint navigation',
    });
    expect(toggle.querySelector('svg')).toBeInTheDocument();
    expect(toggle).toHaveTextContent('Expand');
    expect(
      JSON.parse(window.localStorage.getItem('rpd::__ui_sidenav')!),
    ).toEqual({ endpoint: false });
  });

  it('persists endpoint navigation independently from the other sidenavs', async () => {
    const user = userEvent.setup();
    window.localStorage.setItem(
      'rpd::__ui_sidenav',
      JSON.stringify({ main: false, assistant: true, endpoint: false }),
    );

    render(
      <MemoryRouter
        initialEntries={['/deployment/endpoint/endpoint-1/overview']}
      >
        <EndpointSideNav endpointId="endpoint-1" />
      </MemoryRouter>,
    );

    expect(
      screen.getByRole('button', { name: 'Expand endpoint navigation' }),
    ).toBeInTheDocument();
    await user.click(
      screen.getByRole('button', { name: 'Expand endpoint navigation' }),
    );
    expect(
      screen.getByRole('button', { name: 'Collapse endpoint navigation' }),
    ).toBeInTheDocument();
    expect(
      JSON.parse(window.localStorage.getItem('rpd::__ui_sidenav')!),
    ).toEqual({ main: false, assistant: true, endpoint: true });
  });

  it('starts collapsed on a small viewport', () => {
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest.fn().mockReturnValue({
        matches: true,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      }),
    });

    render(
      <MemoryRouter
        initialEntries={['/deployment/endpoint/endpoint-1/overview']}
      >
        <EndpointSideNav endpointId="endpoint-1" />
      </MemoryRouter>,
    );

    const navigation = screen.getByLabelText('Endpoint actions');
    expect(navigation.parentElement).toHaveClass('w-12');
    expect(
      screen.getByRole('button', { name: 'Expand endpoint navigation' }),
    ).toBeInTheDocument();
  });

  it('reconciles a small viewport before attaching the media listener', async () => {
    const desktopMedia = {
      matches: false,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
    };
    const mobileMedia = {
      matches: true,
      addEventListener: jest.fn(),
      removeEventListener: jest.fn(),
    };
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: jest
        .fn()
        .mockReturnValueOnce(desktopMedia)
        .mockReturnValue(mobileMedia),
    });

    render(
      <MemoryRouter
        initialEntries={['/deployment/endpoint/endpoint-1/versions']}
      >
        <EndpointSideNav endpointId="endpoint-1" />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(
        screen.getByLabelText('Endpoint actions').parentElement,
      ).toHaveClass('w-12');
    });
  });
});
