import { useCallback, useContext, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { SocialButtonGroup } from '@/app/components/ui/primitives/buttons/social-button-group';
import { useNavigate, useLocation, useParams } from 'react-router-dom';
import { AuthenticateResponse, ServiceError } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useRapidaStore } from '@/stores/app';
import { AuthContext } from '@/context/auth-context';
import { useWorkspace } from '@/workspace';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  FluidPasswordInput,
  FluidTextInput,
  PrimaryButton,
  Text,
} from '@/app/components/ui/primitives';
import { Notification } from '@/app/components/ui/feedback';
import { ArrowRight } from '@carbon/icons-react';
import { Link } from '@carbon/react';
import { useTheme } from '@/theme/theme-provider';
import { registerUser } from '@/clients';

interface CustomizedState {
  email: string;
}

interface SignUpFormValues {
  name: string;
  email: string;
  password: string;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function SignUpPage() {
  const workspace = useWorkspace();
  const { theme } = useTheme();
  const { setAuthentication } = useContext(AuthContext);
  const location = useLocation();
  const locationState = location?.state as CustomizedState;
  const navigator = useGlobalNavigation();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const navigate = useNavigate();
  const {
    register,
    handleSubmit,
    resetField,
    setFocus,
    setValue,
    formState: { errors },
  } = useForm<SignUpFormValues>({ mode: 'onBlur' });
  const [error, setError] = useState('');
  const { next } = useParams();
  const providers = workspace.authentication.signUp.providers;
  const hasAlternateSignUp =
    providers.google || providers.linkedin || providers.github;

  useEffect(() => {
    if (locationState?.email) setValue('email', locationState.email);
  }, [locationState, setValue]);

  const afterRegisterUser = useCallback(
    (err: ServiceError | null, auth: AuthenticateResponse | null) => {
      hideLoader();
      if (auth?.getSuccess()) {
        setAuthentication?.(auth.getData(), () => {
          navigate(next || '/dashboard');
        });
        return;
      }

      const message = auth?.getError()?.getHumanmessage();
      if (!message) console.error(err);
      setError(message || "We couldn't create your account. Try again.");
      resetField('password');
      setFocus('email');
    },
    [hideLoader, navigate, next, resetField, setAuthentication, setFocus],
  );

  const onRegisterUser = (data: SignUpFormValues) => {
    setError('');
    showLoader('overlay');
    registerUser(data.email, data.password, data.name, afterRegisterUser);
  };

  if (!workspace.authentication.signUp.enable) {
    return (
      <div
        className="flex w-full max-w-[37rem] flex-col bg-surface"
        data-testid="signup-disabled-panel"
      >
        <header className="p-6 sm:p-8">
          <Text as="h1" variant="heading-05">
            403
          </Text>
          <Text as="h2" variant="heading-03" className="mt-4">
            Account creation unavailable
          </Text>
          <Text as="p" variant="body-01" className="mt-2 text-muted">
            Account creation is disabled for this workspace. Contact your
            administrator for assistance.
          </Text>
        </header>
        <div className="p-6 pt-0 sm:p-8 sm:pt-0">
          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            onClick={() => navigator.goTo('/')}
            className="w-full! max-w-none! justify-between!"
          >
            Go to log in
          </PrimaryButton>
        </div>
      </div>
    );
  }

  return (
    <div
      className="flex w-full max-w-[37rem] flex-col bg-surface"
      data-testid="signup-panel"
    >
      <Helmet title="Create your account" />
      <header className="p-6 sm:p-8" data-testid="signup-header">
        <Text as="h1" variant="heading-05" className="m-0">
          Create account
        </Text>
        <Text as="p" variant="body-02" className="mt-2 text-muted">
          Already have an account?{' '}
          <Link href="/auth/signin" inline>
            Log in
          </Link>
        </Text>
      </header>

      <form
        noValidate
        className="p-0"
        data-testid="signup-form"
        onSubmit={handleSubmit(onRegisterUser)}
      >
        <div
          className="mx-6 flex flex-col sm:mx-8"
          data-testid="signup-input-stack"
        >
          <FluidTextInput
            id="signup-name"
            labelText="Full name"
            type="text"
            required
            autoComplete="name"
            placeholder="Jane Doe"
            invalid={Boolean(errors.name)}
            invalidText={errors.name?.message}
            {...register('name', { required: 'Enter your full name.' })}
          />
          <FluidTextInput
            id="signup-email"
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
            id="signup-password"
            labelText="Password"
            required
            autoComplete="new-password"
            invalid={Boolean(errors.password)}
            invalidText={errors.password?.message}
            {...register('password', { required: 'Enter a password.' })}
          />
        </div>

        <div className="p-6 sm:p-8" data-testid="signup-actions">
          <Text as="p" variant="legal-01" className="text-muted">
            By creating an account, you agree to the{' '}
            <Link href={theme.links.terms} target="_blank" inline>
              Terms and conditions
            </Link>{' '}
            and{' '}
            <Link href={theme.links.privacy} target="_blank" inline>
              Privacy policy
            </Link>
            .
          </Text>

          {error && (
            <div className="mt-4">
              <Notification
                kind="error"
                title="Unable to create account"
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
            Create account
          </PrimaryButton>
        </div>
      </form>

      {hasAlternateSignUp && (
        <section
          aria-labelledby="alternate-signup-heading"
          className="p-6 sm:p-8"
          data-testid="signup-alternates"
        >
          <Text
            as="p"
            variant="body-02"
            id="alternate-signup-heading"
            className="mb-4 text-muted"
          >
            Other ways to create an account
          </Text>
          <SocialButtonGroup {...providers} actionLabel="Sign up" />
        </section>
      )}
    </div>
  );
}
