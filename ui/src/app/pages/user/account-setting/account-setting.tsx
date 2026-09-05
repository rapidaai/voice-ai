import { RedNoticeBlock } from '@/app/components/container/message/notice-block';
import { FormLabel } from '@/app/components/form-label';
import { PrimaryButton } from '@/app/components/carbon/button';
import { ChevronRight } from '@carbon/icons-react';
import { FieldSet } from '@/app/components/form/fieldset';
import { Input } from '@/app/components/form/input';
import { InputHelper } from '@/app/components/input-helper';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { SectionDivider } from '@/app/components/blocks/section-divider';
import { connectionConfig } from '@/configs';
import { useRapidaStore } from '@/hooks';
import {
  ChangePassword,
  ChangePasswordRequest,
  ConnectionConfig,
} from '@rapidaai/react';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import toast from 'react-hot-toast/headless';
import { useNavigate } from 'react-router-dom';
import { useCurrentCredential } from '@/hooks/use-credential';

export const AccountSetting = () => {
  /**
   * loggedin user
   */
  const { user, token, authId, projectId } = useCurrentCredential();
  /**
   * page error
   */
  const [error, setError] = useState('');

  /**
   * form handling
   */
  const { register, handleSubmit } = useForm();

  /**
   * common loader
   */
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * To naviagate to dashboard
   */
  let navigate = useNavigate();

  /**
   * calling for authentication
   * @param data
   */
  const onChangePassword = data => {
    setError('');
    if (data.password !== data.re_password) {
      setError('The new passwords do not match. Please try again.');
      return;
    }
    showLoader();
    const request = new ChangePasswordRequest();
    request.setOldpassword(data.current_password);
    request.setPassword(data.password);
    ChangePassword(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    )
      .then(rlp => {
        hideLoader();
        if (rlp?.getSuccess()) {
          toast.success(
            'The password has been successfully changed. You will be redirected to the sign-in page.',
          );
          return navigate('/auth/signin');
        } else {
          let errorMessage = rlp?.getError();
          if (errorMessage) setError(errorMessage.getHumanmessage());
          else {
            setError('Unable to process your request. please try again later.');
          }
          return;
        }
      })
      .catch(e => {
        setError('Unable to process your request. please try again later.');
        hideLoader();
      });
  };

  return (
    <form
      id="account-settings-form"
      className="w-full flex flex-col flex-1 min-h-0"
      onSubmit={handleSubmit(onChangePassword)}
    >
      <div className="overflow-auto flex flex-col flex-1">
        <div className="px-4 pt-4 pb-12 flex flex-col gap-10 max-w-2xl">
          <div className="flex flex-col gap-6">
            <SectionDivider label="Account Information" />
            <FieldSet>
              <FormLabel>Name</FormLabel>
              <Input
                disabled
                value={user?.name}
                placeholder="eg: John Deo"
              ></Input>
            </FieldSet>
            <FieldSet>
              <FormLabel>Email</FormLabel>
              <Input
                disabled
                value={user?.email}
                placeholder="eg: john@example.com"
              ></Input>
            </FieldSet>
          </div>
          <div className="flex flex-col gap-6">
            <SectionDivider label="Password" />
            <FieldSet>
              <Input
                name="username"
                required
                type="hidden"
                value={user?.email}
              ></Input>
              <FormLabel>Current Password</FormLabel>
              <Input
                required
                type="password"
                autoComplete=""
                {...register('current_password')}
                placeholder="*******"
              ></Input>
              <InputHelper>Enter your current password to confirm.</InputHelper>
            </FieldSet>
            <FieldSet>
              <FormLabel>New Password</FormLabel>
              <Input
                required
                autoComplete="new-password"
                type="password"
                {...register('password')}
                placeholder="*******"
              ></Input>
            </FieldSet>
            <FieldSet>
              <FormLabel>Confirm Password</FormLabel>
              <Input
                required
                type="password"
                autoComplete="new-password"
                {...register('re_password')}
                placeholder="*******"
              ></Input>
            </FieldSet>
          </div>
          <div className="flex-col gap-6">
            <PrimaryButton
              size="md"
              renderIcon={ChevronRight}
              type="submit"
              form="account-settings-form"
              isLoading={loading}
            >
              Save changes
            </PrimaryButton>
          </div>
        </div>
      </div>
    </form>
  );
};
