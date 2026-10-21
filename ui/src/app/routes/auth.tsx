import {
  AuthSignInPage,
  AuthSignUpPage,
  AuthForgotPasswordPage,
  AuthChangePasswordPage,
} from '@/app/pages/authentication';
import { AuthenticationLayout } from '@/app/components/layout/authentication/authentication-layout';
import { IgnoreBox } from '@/app/components/layout/container/protected-box';
import { Route, Routes } from 'react-router-dom';

export function AuthRoute() {
  return (
    <Routes>
      <Route
        element={
          <IgnoreBox>
            <AuthenticationLayout />
          </IgnoreBox>
        }
      >
        <Route path="signup" element={<AuthSignUpPage />} />
        <Route index element={<AuthSignInPage />} />
        <Route path="signin" element={<AuthSignInPage />} />
        <Route path="forgot-password" element={<AuthForgotPasswordPage />} />
        <Route
          path="change-password/:token"
          element={<AuthChangePasswordPage />}
        />
      </Route>
    </Routes>
  );
}
