import React from 'react';
import '@testing-library/jest-dom';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthRoute } from '../auth';

jest.mock('@/app/pages/authentication', () => ({
  AuthSignInPage: () => <div>Sign in route</div>,
  AuthSignUpPage: () => <div>Sign up route</div>,
  AuthForgotPasswordPage: () => <div>Forgot password route</div>,
  AuthChangePasswordPage: () => <div>Change password route</div>,
}));

jest.mock(
  '@/app/components/layout/authentication/authentication-layout',
  () => {
    const { Outlet } = jest.requireActual('react-router-dom');
    return { AuthenticationLayout: () => <Outlet /> };
  },
);

jest.mock('@/app/components/layout/container/protected-box', () => ({
  IgnoreBox: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

const renderAuthRoute = (initialEntry: string) =>
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/auth/*" element={<AuthRoute />} />
      </Routes>
    </MemoryRouter>,
  );

describe('AuthRoute', () => {
  it.each([
    ['/auth', 'Sign in route'],
    ['/auth/signin', 'Sign in route'],
    ['/auth/signup', 'Sign up route'],
    ['/auth/forgot-password', 'Forgot password route'],
    ['/auth/change-password/token-1', 'Change password route'],
  ])('renders %s', (path, expectedText) => {
    renderAuthRoute(path);

    expect(screen.getByText(expectedText)).toBeInTheDocument();
  });
});
