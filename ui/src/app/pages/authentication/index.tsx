/**
 * Asynchronously loads the component for Signup
 */

import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const AuthSignUpPage = lazyLoad(
  () => import('./sign-up'),
  module => module.SignUpPage,
  {
    fallback: <PageLoader />,
  },
);

export const AuthSignInPage = lazyLoad(
  () => import('./sign-in'),
  module => module.SignInPage,
  {
    fallback: <PageLoader />,
  },
);

export const AuthForgotPasswordPage = lazyLoad(
  () => import('./forgot-password'),
  module => module.ForgotPasswordPage,
  {
    fallback: <PageLoader />,
  },
);

export const AuthChangePasswordPage = lazyLoad(
  () => import('./change-password'),
  module => module.ChangePasswordPage,
  {
    fallback: <PageLoader />,
  },
);
