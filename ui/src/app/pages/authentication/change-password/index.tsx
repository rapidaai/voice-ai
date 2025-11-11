import React, { useCallback, useState } from 'react';
import { DescriptiveHeading } from '@/app/components/Heading/DescriptiveHeading';
import { Helmet } from '@/app/components/Helmet';
import { Input } from '@/app/components/Form/Input';
import { Label } from '@/app/components/Form/Label';
import { useNavigate, useParams } from 'react-router-dom';
import { CreatePassword } from '@rapidaai/react';
import { CreatePasswordResponse } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { Button } from '@/app/components/Form/Button';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { ServiceError } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

/**
 *
 * @returns
 */
export function ChangePasswordPage() {
  /**
   * Form handling
   */
  const { register, handleSubmit } = useForm();
  const [error, setError] = useState('');

  /**
   * navigate
   */
  let navigate = useNavigate();

  /**
   * token
   */
  let { token } = useParams();

  /**
   * after changing the password
   */
  const afterCreatePassword = useCallback(
    (err: ServiceError | null, cpr: CreatePasswordResponse | null) => {
      if (err) {
        setError('unable to process your request. please try again later.');
        return;
      }
      if (cpr?.getSuccess()) {
        return navigate('/auth/signin');
      } else {
        let errorMessage = cpr?.getError();
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
   * @returns
   */
  const onCreatePassword = data => {
    if (!token) {
      setError('Password is invalid or expired, please try again.');
      return;
    }
    if (data.password !== data.confirmPassword) {
      setError('Passwords entered do not match');
      return;
    }
    CreatePassword(connectionConfig, token, data.password, afterCreatePassword);
  };

  return (
    <>
      <Helmet title="Forgot your password"></Helmet>
      <DescriptiveHeading
        heading="Change Password"
        subheading="Don’t worry! Fill in your email and we’ll send you a link to reset your password."
      ></DescriptiveHeading>
      <form
        className="space-y-6 mt-6"
        onSubmit={handleSubmit(onCreatePassword)}
      >
        <fieldset className="mt-2 space-y-1">
          <Label for="password" text="Password"></Label>
          <Input
            {...register('password')}
            type="password"
            placeholder="********"
          ></Input>
        </fieldset>
        <fieldset className="mt-2 space-y-1">
          <Label for="password" text="Re-enter Password"></Label>
          <Input
            {...register('confirmPassword')}
            type="password"
            placeholder="********"
          ></Input>
        </fieldset>
        <ErrorMessage message={error} />
        <fieldset>
          <Button type="submit">Change Password</Button>
        </fieldset>
      </form>
    </>
  );
}
