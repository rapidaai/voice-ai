import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Link, MemoryRouter, Route, Routes } from 'react-router-dom';
import ReactGA from 'react-ga4';
import { GA, getAnalyticsPagePath, isGoogleAnalyticsEnabled } from '../ga';

jest.mock('react-ga4', () => ({
  initialize: jest.fn(),
  send: jest.fn(),
}));

const renderTracker = (measurementId: string) =>
  render(
    <MemoryRouter initialEntries={['/dashboard?tab=overview']}>
      <GA
        env="production"
        hostname="app.rapida.ai"
        measurementId={measurementId}
      />
      <Routes>
        <Route
          path="*"
          element={<Link to="/settings?panel=billing">Go to settings</Link>}
        />
      </Routes>
    </MemoryRouter>,
  );

describe('GA', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('sends page views in production and keeps initialization one-time', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    renderTracker('G-APP-SHELL-1');

    await waitFor(() =>
      expect(ReactGA.initialize).toHaveBeenCalledWith('G-APP-SHELL-1'),
    );
    expect(ReactGA.initialize).toHaveBeenCalledTimes(1);
    expect(ReactGA.send).toHaveBeenLastCalledWith({
      hitType: 'pageview',
      page: '/dashboard?tab=overview',
    });

    await user.click(screen.getByRole('link', { name: 'Go to settings' }));

    await waitFor(() =>
      expect(ReactGA.send).toHaveBeenLastCalledWith({
        hitType: 'pageview',
        page: '/settings?panel=billing',
      }),
    );
    expect(ReactGA.initialize).toHaveBeenCalledTimes(1);
  });

  it('does not send analytics outside production or on local hosts', () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <GA env="development" hostname="app.rapida.ai" measurementId="G-DEV" />
        <GA env="production" hostname="localhost" measurementId="G-LOCAL" />
      </MemoryRouter>,
    );

    expect(ReactGA.initialize).not.toHaveBeenCalled();
    expect(ReactGA.send).not.toHaveBeenCalled();
  });

  it('detects analytics eligibility and formats routed page paths', () => {
    expect(
      isGoogleAnalyticsEnabled({
        env: 'production',
        hostname: 'app.rapida.ai',
        measurementId: 'G-TEST',
      }),
    ).toBe(true);
    expect(
      isGoogleAnalyticsEnabled({
        env: 'production',
        hostname: '127.0.0.1',
        measurementId: 'G-TEST',
      }),
    ).toBe(false);
    expect(
      isGoogleAnalyticsEnabled({
        env: 'production',
        hostname: 'app.rapida.ai',
        measurementId: '',
      }),
    ).toBe(false);
    expect(
      getAnalyticsPagePath({
        pathname: '/logs',
        search: '?source=endpoint',
      }),
    ).toBe('/logs?source=endpoint');
  });
});
