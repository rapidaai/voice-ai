import { useCallback, useContext, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/helmet';
import { SocialButtonGroup } from '@/app/components/carbon/button/social-button-group';
import { useNavigate, useLocation } from 'react-router-dom';
import { RegisterUser } from '@rapidaai/react';
import { AuthenticateResponse } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import { useRapidaStore } from '@/hooks';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import { useWorkspace } from '@/workspace';
import { connectionConfig } from '@/configs';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { Stack, TextInput } from '@/app/components/carbon/form';
import { PrimaryButton } from '@/app/components/carbon/button';
import { Notification } from '@/app/components/carbon/notification';
import { ArrowRight } from '@carbon/icons-react';
import { Link, PasswordInput } from '@carbon/react';
import { useTheme } from '@/theme/theme-provider';

interface CustomizedState {
  email: string;
}

export function SignUpPage() {
  const workspace = useWorkspace();
  const { theme } = useTheme();
  const { setAuthentication } = useContext(AuthContext);
  const location = useLocation();
  const locationState = location?.state as CustomizedState;
  const navigator = useGlobalNavigation();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  let navigate = useNavigate();
  const { register, handleSubmit, setValue } = useForm();

  useEffect(() => {
    if (locationState?.email) setValue('email', locationState.email);
  }, [locationState]);

  const [error, setError] = useState('');
  let { next } = useParams();

  const afterRegisterUser = useCallback(
    (err: ServiceError | null, auth: AuthenticateResponse | null) => {
      hideLoader();
      if (auth?.getSuccess()) {
        let at = auth.getData();
        if (setAuthentication)
          setAuthentication(at, () => {
            if (next) return navigate(next);
            return navigate('/dashboard');
          });
      } else {
        let errorMessage = auth?.getError();
        if (errorMessage) setError(errorMessage.getHumanmessage());
        else
          setError('Unable to process your request. please try again later.');
        return;
      }
    },
    [],
  );

  const onRegisterUser = data => {
    showLoader('overlay');
    RegisterUser(
      connectionConfig,
      data.email,
      data.password,
      data.name,
      afterRegisterUser,
    );
  };

  if (!workspace.authentication.signUp.enable) {
    return (
      <div className="flex flex-1">
        <div className="max-w-md">
          <h1 className="text-3xl font-light tracking-tight">403</h1>
          <p className="text-2xl font-light tracking-tight mt-4">
            Sign-up not enabled
          </p>
          <p className="mb-8 mt-2 text-sm text-gray-500 dark:text-gray-400">
            Sign-up is currently disabled for this workspace. Please contact
            your administrator for assistance.
          </p>
          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            onClick={() => navigator.goTo('/')}
            className="!w-full !max-w-none !justify-between"
          >
            Go to signin
          </PrimaryButton>
        </div>
      </div>
    );
  }

  return (
    <>
      <Helmet title="Sign up to your account" />
      <Stack gap={6}>
        <Stack gap={2}>
          <h1 className="m-0 text-[1.8rem] leading-tight">Signup</h1>
          <p className="mt-1.5 text-sm leading-[1.4286] text-(--cds-text-secondary)">
            Already have an account? &nbsp;
            <Link href="/auth/signin" className="text-sm">
              Sign-in
            </Link>
          </p>
          <div
            aria-hidden="true"
            className={`h-px bg-gray-300 dark:bg-gray-900 mt-3`}
          />
        </Stack>
        <form onSubmit={handleSubmit(onRegisterUser)}>
          <Stack gap={5}>
            <TextInput
              id="signup-name"
              labelText="Name"
              type="text"
              required
              autoComplete="name"
              placeholder="eg: John Doe"
              {...register('name', { required: 'Please enter your name' })}
            />
            <TextInput
              id="signup-email"
              labelText="Email Address"
              type="email"
              required
              autoComplete="email"
              placeholder="eg: john@example.com"
              {...register('email', { required: 'Please enter email' })}
            />
            <PasswordInput
              id="signup-password"
              labelText="Password"
              required
              autoComplete="new-password"
              placeholder="********"
              {...register('password', { required: 'Please enter password' })}
            />
            {error && (
              <Notification kind="error" title="Error" subtitle={error} />
            )}
            <PrimaryButton
              size="lg"
              renderIcon={ArrowRight}
              isLoading={loading}
              type="submit"
              className="!w-full !max-w-none !justify-between"
            >
              Continue
            </PrimaryButton>
          </Stack>
          <div
            aria-hidden="true"
            className={`h-px bg-gray-300 dark:bg-gray-900 mt-3`}
          />
        </form>
        <Stack gap={2}>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            By signing up, you agree to the{' '}
            <Link href={theme.links.terms} target="_blank" inline>
              Terms and Conditions
            </Link>{' '}
            and{' '}
            <Link href={theme.links.privacy} target="_blank" inline>
              Privacy Policy
            </Link>
            .
          </p>
        </Stack>
        <SocialButtonGroup {...workspace.authentication.signIn.providers} />
      </Stack>
    </>
  );
}
