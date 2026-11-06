import {
  AuthSignInPage,
  AuthSignUpPage,
  AuthForgotPasswordPage,
  AuthChangePasswordPage,
} from '@/app/pages/authentication';
import { AuthenticationLayout } from '@/app/components/layout/authentication/authentication-layout';
import { IgnoreBox } from '@/app/components/layout/container/protected-box';
import type { RouteObject } from 'react-router-dom';

export const authRoute: RouteObject = {
  path: '/auth',
  children: [
    {
      element: (
        <IgnoreBox>
          <AuthenticationLayout />
        </IgnoreBox>
      ),
      children: [
        { index: true, element: <AuthSignInPage /> },
        { path: 'signin', element: <AuthSignInPage /> },
        { path: 'signup', element: <AuthSignUpPage /> },
        { path: 'forgot-password', element: <AuthForgotPasswordPage /> },
        {
          path: 'change-password/:token',
          element: <AuthChangePasswordPage />,
        },
      ],
    },
  ],
};
