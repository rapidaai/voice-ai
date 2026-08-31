import React, { useCallback, useState } from 'react';
import { Helmet } from '@/app/components/helmet';
import { ForgotPassword } from '@rapidaai/react';
import { ForgotPasswordResponse } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useRapidaStore } from '@/hooks';
import { ServiceError } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { Stack, TextInput } from '@/app/components/carbon/form';
import { PrimaryButton } from '@/app/components/carbon/button';
import { Notification } from '@/app/components/carbon/notification';
import { ArrowRight } from '@carbon/icons-react';
import { Link } from '@carbon/react';
import { useTheme } from '@/theme/theme-provider';

export function ForgotPasswordPage() {
  const { theme } = useTheme();
  const { register, handleSubmit } = useForm();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [error, setError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const supportContact = theme.links.support.replace(/^mailto:/, '');

  const afterForgotPassword = useCallback(
    (err: ServiceError | null, fpr: ForgotPasswordResponse | null) => {
      hideLoader();
      if (err) {
        setError('Unable to process your request. Please try again later.');
      }
      if (fpr?.getSuccess()) {
        setError('');
        setSuccessMessage(
          `Thanks! An email was sent that will ask you to click on a link to verify that you own this account. If you don't get the email, please contact support at ${supportContact}.`,
        );
      } else {
        let errorMessage = fpr?.getError();
        if (errorMessage) setError(errorMessage.getHumanmessage());
        else
          setError('Unable to process your request. Please try again later.');
        return;
      }
    },
    [hideLoader, supportContact],
  );

  const onForgotPassword = data => {
    showLoader('overlay');
    ForgotPassword(connectionConfig, data.email, afterForgotPassword);
  };

  return (
    <Stack gap={6}>
      <Helmet title="Forgot your password" />
      <Stack gap={2}>
        <h1 className="m-0 text-[1.8rem] leading-tight">Forgot Password</h1>
        <div
          aria-hidden="true"
          className={`h-px bg-gray-300 dark:bg-gray-900 mt-3`}
        />
      </Stack>
      <form className="mt-6" onSubmit={handleSubmit(onForgotPassword)}>
        <Stack gap={5}>
          <TextInput
            id="forgot-email"
            labelText="Email Address"
            type="email"
            required
            autoComplete="email"
            disabled={loading}
            placeholder="eg: john@example.com"
            {...register('email')}
          />
          {error && (
            <Notification kind="error" title="Error" subtitle={error} />
          )}
          {successMessage && (
            <Notification
              kind="success"
              title="Email sent"
              subtitle={successMessage}
            />
          )}
          <PrimaryButton
            size="lg"
            renderIcon={ArrowRight}
            type="submit"
            isLoading={loading}
            className="!w-full !max-w-none !justify-between"
          >
            Send Email
          </PrimaryButton>
        </Stack>
        <div
          aria-hidden="true"
          className={`h-px bg-gray-300 dark:bg-gray-900 mt-3`}
        />
      </form>

      <Stack gap={2}>
        <p className="text-center">
          <Link href="/auth/signin" className="text-sm">
            Back to sign in?
          </Link>
        </p>
      </Stack>
    </Stack>
  );
}
