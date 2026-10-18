import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AuthContext } from '@/context/auth-context';
import { IgnoreBox, ProtectedBox } from './protected-box';

const authenticatedValue = {
  isAuthenticated: () => true,
  isThereOrganization: () => true,
  isThereProject: () => true,
  unauthenticate: () => undefined,
};

const unauthenticatedValue = {
  ...authenticatedValue,
  isAuthenticated: () => false,
};

const meta = {
  title: 'Layout/Container/ProtectedBox',
  component: ProtectedBox,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Route guard container for authenticated workspace pages and auth bypass pages.',
      },
    },
  },
} satisfies Meta<typeof ProtectedBox>;

export default meta;

type Story = StoryObj<typeof meta>;

export const Allowed: Story = {
  args: {
    children: <div>Protected content</div>,
  },
  render: () => (
    <AuthContext.Provider value={authenticatedValue}>
      <MemoryRouter initialEntries={['/dashboard']}>
        <ProtectedBox>
          <div className="p-4 text-sm text-foreground">Protected content</div>
        </ProtectedBox>
      </MemoryRouter>
    </AuthContext.Provider>
  ),
};

export const RedirectToSignIn: Story = {
  args: {
    children: <div>Protected content</div>,
  },
  render: () => (
    <AuthContext.Provider value={unauthenticatedValue}>
      <MemoryRouter initialEntries={['/dashboard?tab=usage']}>
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
            path="/auth/signin"
            element={<div className="p-4 text-sm text-foreground">Sign in</div>}
          />
        </Routes>
      </MemoryRouter>
    </AuthContext.Provider>
  ),
};

export const IgnoredAuthPage: Story = {
  args: {
    children: <div>Auth content</div>,
  },
  render: () => (
    <AuthContext.Provider value={authenticatedValue}>
      <MemoryRouter initialEntries={['/auth/signin']}>
        <IgnoreBox>
          <div className="p-4 text-sm text-foreground">Auth content</div>
        </IgnoreBox>
      </MemoryRouter>
    </AuthContext.Provider>
  ),
};
