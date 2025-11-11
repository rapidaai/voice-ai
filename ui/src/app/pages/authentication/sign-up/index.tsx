import { useCallback, useContext, useEffect, useState } from 'react';
import { FormActionHeading } from '@/app/components/Heading/ActionHeading/FormActionHeading';
import { Helmet } from '@/app/components/Helmet';
import { SocialButtonGroup } from '@/app/components/Form/ButtonGroup/SocialButtonGroup';
import { IBlueBGArrowButton } from '@/app/components/Form/Button';
import { Input } from '@/app/components/Form/Input';
import { useNavigate, useLocation } from 'react-router-dom';
import { RegisterUser } from '@rapidaai/react';
import { AuthenticateResponse } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useParams } from 'react-router-dom';
import { useRapidaStore } from '@/hooks';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { ServiceError } from '@rapidaai/react';
import { AuthContext } from '@/context/auth-context';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { useWorkspace } from '@/context/workplace-context';
import { connectionConfig } from '@/configs';

/**
 * External state get passed
 */
interface CustomizedState {
  email: string;
}

export function SignUpPage() {
  /**
   * setting up authentication after creation of user
   */
  const workspace = useWorkspace();
  const { setAuthentication } = useContext(AuthContext);
  const location = useLocation();
  const locationState = location?.state as CustomizedState;
  /**
   * loading context
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * To naviagate to dashboard
   */
  let navigate = useNavigate();

  /**
   * form controlling
   */
  const { register, handleSubmit, setValue } = useForm();

  /**
   * what if email get passed from home page
   */
  useEffect(() => {
    if (locationState?.email) setValue('email', locationState.email);
  }, [locationState]);

  const [error, setError] = useState('');

  //   redirectinng when needed
  let { next } = useParams();

  /**
   * callback after registering user
   */
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

  /**
   * element
   */

  return (
    <>
      <Helmet title="Signing up to your account"></Helmet>
      <FormActionHeading
        heading="Sign up"
        action={
          <a
            className="underline leading-6 text-blue-600 hover:text-blue-500"
            href="/auth/signin"
          >
            I already have an account
          </a>
        }
      ></FormActionHeading>
      <form className="space-y-6 mt-6" onSubmit={handleSubmit(onRegisterUser)}>
        <FieldSet>
          <FormLabel>Name</FormLabel>
          <Input
            autoComplete="name"
            type="text"
            required
            className="bg-light-background"
            placeholder="eg: John deo"
            {...register('name', {
              required: 'Please enter your name',
            })}
          ></Input>
        </FieldSet>
        <FieldSet>
          <FormLabel>Email Address</FormLabel>
          <Input
            autoComplete="email"
            type="email"
            required
            className="bg-light-background"
            placeholder="eg: john@rapida.ai"
            {...register('email', {
              required: 'Please enter email',
            })}
          ></Input>
        </FieldSet>
        <FieldSet>
          <FormLabel>Password</FormLabel>
          <Input
            autoComplete="password"
            type="password"
            required
            className="bg-light-background"
            placeholder="********"
            {...register('password', {
              required: 'Please enter password',
            })}
          ></Input>
        </FieldSet>

        <ErrorMessage message={error} />
        <IBlueBGArrowButton
          className="w-full justify-between"
          type="submit"
          isLoading={loading}
        >
          Continue
        </IBlueBGArrowButton>
      </form>
      <div className="mt-4 space-y-4">
        <FieldSet className="text-sm">
          <p className="text-gray-600 dark:text-gray-400">
            By signing up, you agree to the &nbsp;
            <a
              className="underline font-semibold"
              target="_blank"
              href="/static/terms-conditions"
            >
              Terms and Conditions
            </a>
            &nbsp; and &nbsp;
            <a
              href="/static/privacy-policy"
              target="_blank"
              className="underline font-semibold"
            >
              Privacy Policy
            </a>
            .
          </p>
        </FieldSet>
        <SocialButtonGroup
          {...workspace.authentication.signIn.providers}
        ></SocialButtonGroup>
      </div>
    </>
  );
}
