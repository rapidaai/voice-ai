import React, { useCallback, useState } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { Input } from '@/app/components/Form/Input';
import { ForgotPassword } from '@rapidaai/react';
import { ForgotPasswordResponse } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useRapidaStore } from '@/hooks';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { IBlueBGArrowButton } from '@/app/components/Form/Button';
import { ServiceError } from '@rapidaai/react';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { MoveRight } from 'lucide-react';
import { Spinner } from '@/app/components/Loader/Spinner';
import { FormActionHeading } from '@/app/components/Heading/ActionHeading/FormActionHeading';
import { connectionConfig } from '@/configs';

export function ForgotPasswordPage() {
  /**
   * handling the form submission
   */
  const { register, handleSubmit } = useForm();

  /**
   * loading
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * error and success message
   */
  const [error, setError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');

  /**
   * after sending email for forgot password
   */

  const afterForgotPassword = useCallback(
    (err: ServiceError | null, fpr: ForgotPasswordResponse | null) => {
      hideLoader();
      if (err) {
        setError('unable to process your request. please try again later');
      }
      if (fpr?.getSuccess()) {
        setError('');
        setSuccessMessage(
          "Thanks! An email was sent that will ask you to click on a link to verify that you own this account. If you don't get the email, please contact support@rapida.ai.",
        );
        //   return redirectToOnboarding();
      } else {
        let errorMessage = fpr?.getError();
        if (errorMessage) setError(errorMessage.getHumanmessage());
        else
          setError('Unable to process your request. please try again later.');
        return;
      }
    },
    [],
  );
  /**
   *
   * @param data
   */
  const onForgotPassword = data => {
    showLoader('overlay');
    ForgotPassword(connectionConfig, data.email, afterForgotPassword);
  };

  return (
    <>
      <Helmet title="Forgot your password"></Helmet>
      <FormActionHeading heading="Forgot Password"></FormActionHeading>
      <form
        className="space-y-6 mt-6"
        onSubmit={handleSubmit(onForgotPassword)}
      >
        <FieldSet>
          <FormLabel>Email Address</FormLabel>
          <Input
            autoComplete="email"
            type="email"
            required
            disabled={loading}
            className="bg-light-background"
            placeholder="eg: john@rapida.ai"
            {...register('email')}
          ></Input>
        </FieldSet>
        <ErrorMessage message={error} />
        {successMessage && (
          <fieldset>
            <p className="p-4 text-sm text-green-800 rounded-lg bg-green-50 dark:bg-gray-800 dark:text-green-400 font-semibold">
              {successMessage}
            </p>
          </fieldset>
        )}
        <IBlueBGArrowButton
          type="submit"
          className="w-full justify-between"
          isLoading={loading}
        >
          Send Email
        </IBlueBGArrowButton>
      </form>
      <p className="mt-4 text-center text-gray-500">
        <a
          className="leading-6 text-blue-600 hover:text-blue-500 underline"
          href="/auth/signin"
        >
          Back to sign in?
        </a>
      </p>
    </>
  );
}
