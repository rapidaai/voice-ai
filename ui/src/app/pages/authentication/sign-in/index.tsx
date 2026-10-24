import { useCallback, useContext, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { SocialButtonGroup } from '@/app/components/ui/primitives/buttons/social-button-group';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useForm } from 'react-hook-form';
import { AuthenticateResponse, ServiceError } from '@rapidaai/react';
import { useRapidaStore } from '@/stores/app';
import { AuthContext } from '@/context/auth-context';
import { useWorkspace } from '@/workspace';
import {
  FluidPasswordInput,
  FluidTextInput,
  Text,
} from '@/app/components/ui/primitives';
import { PrimaryButton } from '@/app/components/ui/primitives';
import { Notification } from '@/app/components/ui/feedback';
import { ArrowRight } from '@carbon/icons-react';
import { Link } from '@carbon/react';
import {
  authenticateUser,
  githubAuth,
  googleAuth,
  linkedinAuth,
} from '@/clients';

interface SignInFormValues {
  email: string;
  password: string;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function SignInPage() {
  const navigate = useNavigate();
  const { setAuthentication } = useContext(AuthContext);
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const {
    register,
    handleSubmit,
    resetField,
    setFocus,
    formState: { errors },
  } = useForm<SignInFormValues>({ mode: 'onBlur' });
  const [error, setError] = useState('');
  const workspace = useWorkspace();
  const [searchParams] = useSearchParams();
  const { next, externalValidation, code, state } = Object.fromEntries(
    searchParams.entries(),
  );
  const providers = workspace.authentication.signIn.providers;
  const hasAlternateLogin =
    providers.google || providers.linkedin || providers.github;

  const afterAuthenticate = useCallback(
    (err: ServiceError | null, auth: AuthenticateResponse | null) => {
      hideLoader();
      if (auth?.getSuccess()) {
        setAuthentication?.(auth.getData(), () => {
          if (next && externalValidation) {
            window.location.replace(next);
            return;
          }
          navigate('/dashboard');
        });
        return;
      }

      const message = auth?.getError()?.getHumanmessage();
      if (!message) console.error(err);
      setError(message || "We couldn't log you in. Try again.");
      resetField('password');
      setFocus('email');
    },
    [
      externalValidation,
      hideLoader,
      navigate,
      next,
      resetField,
      setAuthentication,
      setFocus,
    ],
  );

  const onAuthenticate = (data: SignInFormValues) => {
    setError('');
    showLoader();
    authenticateUser(data.email, data.password, afterAuthenticate);
  };

  useEffect(() => {
    if (state && code) {
      showLoader();
      if (state === 'google') googleAuth(afterAuthenticate, state, code);
      if (state === 'linkedin') linkedinAuth(afterAuthenticate, state, code);
      if (state === 'github') githubAuth(afterAuthenticate, state, code);
    }
  }, [afterAuthenticate, code, showLoader, state]);

  return (
    <div
      className="flex w-full max-w-[37rem] flex-col bg-surface"
      data-testid="signin-panel"
    >
      <Helmet title="Log in to your account" />
      <header className="p-6 sm:p-8" data-testid="signin-header">
        <Text as="h1" variant="heading-05" className="m-0">
          Log in
        </Text>
        {workspace.authentication.signUp.enable && (
          <Text as="p" variant="body-02" className="mt-2 text-muted">
            Don’t have an account?{' '}
            <Link href="/auth/signup" inline>
              Create an account
            </Link>
          </Text>
        )}
      </header>

      <form
        noValidate
        className="p-0"
        data-testid="signin-form"
        onSubmit={handleSubmit(onAuthenticate)}
      >
        <div
          className="mx-6 flex flex-col sm:mx-8"
          data-testid="signin-input-stack"
        >
          <FluidTextInput
            id="signin-email"
            labelText="Email address"
            type="email"
            required
            autoComplete="email"
            placeholder="name@example.com"
            invalid={Boolean(errors.email)}
            invalidText={errors.email?.message}
            {...register('email', {
              required: 'Enter your email address.',
              pattern: {
                value: EMAIL_PATTERN,
                message: 'Enter a valid email address.',
              },
            })}
          />
          <FluidPasswordInput
            id="signin-password"
            labelText="Password"
            required
            autoComplete="current-password"
            invalid={Boolean(errors.password)}
            invalidText={errors.password?.message}
            {...register('password', { required: 'Enter your password.' })}
          />
        </div>

        <div className="p-6 sm:p-8" data-testid="signin-actions">
          <div className="flex justify-end">
            <Link href="/auth/forgot-password" inline>
              <Text variant="body-compact-01">Forgot password?</Text>
            </Link>
          </div>

          {error && (
            <div className="mt-4">
              <Notification
                kind="error"
                title="Unable to log in"
                subtitle={error}
              />
            </div>
          )}

          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            isLoading={loading}
            type="submit"
            className="mt-6! w-full! max-w-none! justify-between!"
          >
            Log in
          </PrimaryButton>
        </div>
      </form>

      {hasAlternateLogin && (
        <section
          aria-labelledby="alternate-login-heading"
          className="p-6 sm:p-8"
          data-testid="signin-alternates"
        >
          <Text
            as="p"
            variant="body-02"
            id="alternate-login-heading"
            className="mb-4 text-muted"
          >
            Other ways to log in
          </Text>
          <SocialButtonGroup {...providers} actionLabel="Log in" />
        </section>
      )}
    </div>
  );
}
