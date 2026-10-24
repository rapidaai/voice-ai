import { useCallback, useState } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { useNavigate, useParams } from 'react-router-dom';
import { CreatePasswordResponse, ServiceError } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useRapidaStore } from '@/stores/app';
import {
  FluidPasswordInput,
  PrimaryButton,
  Text,
} from '@/app/components/ui/primitives';
import { Notification } from '@/app/components/ui/feedback';
import { ArrowRight } from '@carbon/icons-react';
import { Link } from '@carbon/react';
import { createPassword } from '@/clients';

interface ChangePasswordFormValues {
  password: string;
  confirmPassword: string;
}

export function ChangePasswordPage() {
  const {
    register,
    handleSubmit,
    clearErrors,
    setError: setFieldError,
    setFocus,
    formState: { errors },
  } = useForm<ChangePasswordFormValues>({ mode: 'onBlur' });
  const [error, setError] = useState('');
  const navigate = useNavigate();
  const { token } = useParams();
  const { loading, showLoader, hideLoader } = useRapidaStore();

  const afterCreatePassword = useCallback(
    (err: ServiceError | null, response: CreatePasswordResponse | null) => {
      hideLoader();
      if (response?.getSuccess()) {
        navigate('/auth/signin');
        return;
      }

      const message = response?.getError()?.getHumanmessage();
      if (!message) console.error(err);
      setError(message || "We couldn't reset your password. Try again.");
      setFocus('password');
    },
    [hideLoader, navigate, setFocus],
  );

  const onCreatePassword = (data: ChangePasswordFormValues) => {
    setError('');
    clearErrors('confirmPassword');
    if (!token) {
      setError('This reset link has expired. Request a new reset link.');
      return;
    }
    if (data.password !== data.confirmPassword) {
      setFieldError('confirmPassword', {
        type: 'validate',
        message: 'Passwords do not match.',
      });
      setFocus('confirmPassword');
      return;
    }

    showLoader();
    createPassword(token, data.password, afterCreatePassword);
  };

  return (
    <div
      className="flex w-full max-w-[37rem] flex-col bg-surface"
      data-testid="change-password-panel"
    >
      <Helmet title="Set a new password" />
      <header className="p-6 sm:p-8" data-testid="change-password-header">
        <Text as="h1" variant="heading-05" className="m-0">
          Set a new password
        </Text>
        <Text as="p" variant="body-02" className="mt-2 text-muted">
          Choose a new password for your account.
        </Text>
      </header>

      <form
        noValidate
        className="p-0"
        data-testid="change-password-form"
        onSubmit={handleSubmit(onCreatePassword)}
      >
        <div
          className="mx-6 flex flex-col sm:mx-8"
          data-testid="change-password-input-stack"
        >
          <FluidPasswordInput
            id="new-password"
            labelText="New password"
            required
            autoComplete="new-password"
            invalid={Boolean(errors.password)}
            invalidText={errors.password?.message}
            {...register('password', { required: 'Enter a new password.' })}
          />
          <FluidPasswordInput
            id="confirm-password"
            labelText="Confirm password"
            required
            autoComplete="new-password"
            invalid={Boolean(errors.confirmPassword)}
            invalidText={errors.confirmPassword?.message}
            {...register('confirmPassword', {
              required: 'Confirm your new password.',
            })}
          />
        </div>

        <div className="p-6 sm:p-8" data-testid="change-password-actions">
          {error && (
            <div className="mb-6">
              <Notification
                kind="error"
                title="Unable to reset password"
                subtitle={error}
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
            Save new password
          </PrimaryButton>

          <div className="mt-6">
            <Link href="/auth/forgot-password" inline>
              <Text variant="body-compact-01">Request another reset link</Text>
            </Link>
          </div>
        </div>
      </form>
    </div>
  );
}
