import { useCallback, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { ForgotPasswordResponse, ServiceError } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useRapidaStore } from '@/stores/app';
import {
  FluidTextInput,
  PrimaryButton,
  Text,
} from '@/app/components/ui/primitives';
import { Notification } from '@/app/components/ui/feedback';
import { ArrowRight } from '@carbon/icons-react';
import { Link } from '@carbon/react';
import { useTheme } from '@/theme/theme-provider';
import { forgotPassword } from '@/clients';

interface ForgotPasswordFormValues {
  email: string;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function ForgotPasswordPage() {
  const { theme } = useTheme();
  const {
    register,
    handleSubmit,
    setFocus,
    formState: { errors },
  } = useForm<ForgotPasswordFormValues>({ mode: 'onBlur' });
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [error, setError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const supportContact = theme.links.support.replace(/^mailto:/, '');

  const afterForgotPassword = useCallback(
    (err: ServiceError | null, response: ForgotPasswordResponse | null) => {
      hideLoader();
      if (response?.getSuccess()) {
        setError('');
        setSuccessMessage(
          `Thanks! An email was sent that will ask you to click on a link to verify that you own this account. If you don't get the email, please contact support at ${supportContact}.`,
        );
        return;
      }

      const message = response?.getError()?.getHumanmessage();
      if (!message) console.error(err);
      setSuccessMessage('');
      setError(message || "We couldn't send a reset link. Try again.");
    },
    [hideLoader, supportContact],
  );

  useEffect(() => {
    if (!loading && error) setFocus('email');
  }, [error, loading, setFocus]);

  const onForgotPassword = (data: ForgotPasswordFormValues) => {
    setError('');
    setSuccessMessage('');
    showLoader('overlay');
    forgotPassword(data.email, afterForgotPassword);
  };

  return (
    <div
      className="flex w-full max-w-[37rem] flex-col bg-surface"
      data-testid="forgot-password-panel"
    >
      <Helmet title="Reset your password" />
      <header className="p-6 sm:p-8" data-testid="forgot-password-header">
        <Text as="h1" variant="heading-05" className="m-0">
          Reset your password
        </Text>
        <Text as="p" variant="body-02" className="mt-2 text-muted">
          Enter your email and we will send you a reset link.
        </Text>
      </header>

      <form
        noValidate
        className="p-0"
        data-testid="forgot-password-form"
        onSubmit={handleSubmit(onForgotPassword)}
      >
        <div
          className="mx-6 flex flex-col sm:mx-8"
          data-testid="forgot-password-input-stack"
        >
          <FluidTextInput
            id="forgot-email"
            labelText="Email address"
            type="email"
            required
            autoComplete="email"
            disabled={loading}
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
        </div>

        <div className="p-6 sm:p-8" data-testid="forgot-password-actions">
          {error && (
            <div className="mb-6">
              <Notification
                kind="error"
                title="Unable to send reset link"
                subtitle={error}
              />
            </div>
          )}
          {successMessage && (
            <div className="mb-6">
              <Notification
                kind="success"
                title="Email sent"
                subtitle={successMessage}
              />
            </div>
          )}

          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            type="submit"
            isLoading={loading}
            className="w-full! max-w-none! justify-between!"
          >
            Send reset link
          </PrimaryButton>

          <div className="mt-6">
            <Link href="/auth/signin" inline>
              <Text variant="body-compact-01">Back to log in</Text>
            </Link>
          </div>
        </div>
      </form>
    </div>
  );
}
