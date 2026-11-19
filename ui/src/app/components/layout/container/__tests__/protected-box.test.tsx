import React, { useCallback, useState } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthContext } from '@/context/auth-context';
import type { AuthenticationType } from '@/types';
import { IgnoreBox, ProtectedBox } from '../protected-box';

type AuthValue = Partial<AuthenticationType>;

const authenticatedValue: AuthValue = {
  isAuthenticated: () => true,
  hasOrganizations: () => true,
  isThereOrganization: () => true,
  isThereProject: () => true,
};

const renderProtectedRoute = (initialEntry: string, authValue: AuthValue) =>
  render(
    <AuthContext.Provider value={authValue}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route
            path="/dashboard"
            element={
              <ProtectedBox>
                <div>Protected content</div>
              </ProtectedBox>
            }
          />
          <Route
            path="/onboarding/organization"
            element={
              <ProtectedBox>
                <div>Organization onboarding</div>
              </ProtectedBox>
            }
          />
          <Route
            path="/onboarding/project"
            element={
              <ProtectedBox>
                <div>Project onboarding</div>
              </ProtectedBox>
            }
          />
          <Route path="/auth/signin" element={<div>Sign in</div>} />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>,
  );

describe('ProtectedBox', () => {
  it('renders protected content when auth, organization, and project are valid', () => {
    renderProtectedRoute('/dashboard', authenticatedValue);

    expect(screen.getByText('Protected content')).toBeInTheDocument();
  });

  it('redirects unauthenticated users to sign in and preserves search params', () => {
    renderProtectedRoute('/dashboard?tab=usage', {
      ...authenticatedValue,
      isAuthenticated: () => false,
    });

    expect(screen.getByText('Sign in')).toBeInTheDocument();
  });

  it('redirects users without memberships except on organization onboarding', () => {
    renderProtectedRoute('/dashboard', {
      ...authenticatedValue,
      hasOrganizations: () => false,
    });

    expect(screen.getByText('Organization onboarding')).toBeInTheDocument();

    renderProtectedRoute('/onboarding/organization', {
      ...authenticatedValue,
      hasOrganizations: () => false,
    });

    expect(screen.getAllByText('Organization onboarding')).toHaveLength(2);
  });

  it('keeps users with memberships in the shell before tenant selection', () => {
    renderProtectedRoute('/dashboard', {
      ...authenticatedValue,
      isThereOrganization: () => false,
      isThereProject: () => false,
    });

    expect(screen.getByText('Protected content')).toBeInTheDocument();
  });

  it('renders project onboarding when explicitly requested', () => {
    renderProtectedRoute('/onboarding/project', {
      ...authenticatedValue,
      isThereProject: () => false,
    });

    expect(screen.getByText('Project onboarding')).toBeInTheDocument();
  });
});

describe('IgnoreBox', () => {
  it('unauthenticates auth pages without a valid external auth redirect', async () => {
    const unauthenticate = jest.fn();

    render(
      <AuthContext.Provider value={{ unauthenticate }}>
        <MemoryRouter initialEntries={['/auth/signin']}>
          <IgnoreBox>
            <div>Auth content</div>
          </IgnoreBox>
        </MemoryRouter>
      </AuthContext.Provider>,
    );

    expect(screen.getByText('Auth content')).toBeInTheDocument();
    await waitFor(() => expect(unauthenticate).toHaveBeenCalledTimes(1));
  });

  it('renders auth pages when unauthenticate is unavailable', () => {
    render(
      <AuthContext.Provider value={{}}>
        <MemoryRouter initialEntries={['/auth/signin']}>
          <IgnoreBox>
            <div>Auth content</div>
          </IgnoreBox>
        </MemoryRouter>
      </AuthContext.Provider>,
    );

    expect(screen.getByText('Auth content')).toBeInTheDocument();
  });

  it('does not repeat unauthenticate after the auth store updates', async () => {
    const unauthenticate = jest.fn();

    function AuthStoreUpdateHarness() {
      const [, setRefreshCount] = useState(0);
      const onUnauthenticate = useCallback(() => {
        unauthenticate();
        setRefreshCount(count => count + 1);
      }, []);

      return (
        <AuthContext.Provider value={{ unauthenticate: onUnauthenticate }}>
          <MemoryRouter initialEntries={['/auth/signin']}>
            <IgnoreBox>
              <div>Auth content</div>
            </IgnoreBox>
          </MemoryRouter>
        </AuthContext.Provider>
      );
    }

    render(<AuthStoreUpdateHarness />);

    expect(screen.getByText('Auth content')).toBeInTheDocument();
    await waitFor(() => expect(unauthenticate).toHaveBeenCalledTimes(1));
  });

  it('redirects valid external auth sessions without unauthenticating', async () => {
    const replaceLocation = jest.fn();
    const unauthenticate = jest.fn();

    render(
      <AuthContext.Provider
        value={{
          ...authenticatedValue,
          unauthenticate,
        }}
      >
        <MemoryRouter
          initialEntries={[
            '/auth/signin?next=https://example.com/app&externalValidation=1',
          ]}
        >
          <IgnoreBox replaceLocation={replaceLocation}>
            <div>Auth content</div>
          </IgnoreBox>
        </MemoryRouter>
      </AuthContext.Provider>,
    );

    await waitFor(() =>
      expect(replaceLocation).toHaveBeenCalledWith('https://example.com/app'),
    );
    expect(unauthenticate).not.toHaveBeenCalled();
  });
});
