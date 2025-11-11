import {
  AuthSignInPage,
  AuthSignUpPage,
  AuthForgotPasswordPage,
  AuthChangePasswordPage,
} from '@/app/pages/authentication';
import { CenterBox } from '@/app/components/container/center-box';
import { IgnoreBox } from '@/app/components/container/protected-box';
import { Outlet, Route, Routes } from 'react-router-dom';
import { FlexBox } from '@/app/components/container/flex-box';
export function AuthRoute() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <IgnoreBox>
            <FlexBox>
              <CenterBox>
                <Outlet />
              </CenterBox>
            </FlexBox>
          </IgnoreBox>
        }
      >
        <Route path="signup" element={<AuthSignUpPage />} />
        <Route index path="/" element={<AuthSignInPage />} />
        <Route index path="signin" element={<AuthSignInPage />} />
        <Route path="forgot-password" element={<AuthForgotPasswordPage />} />
        <Route
          path="change-password/:token"
          element={<AuthChangePasswordPage />}
        />
      </Route>
    </Routes>
  );
}
