import React, { useState } from 'react';
import { DescriptiveHeading } from '@/app/components/Heading/DescriptiveHeading';
import { Label } from '@/app/components/Form/Label';
import { Input } from '@/app/components/Form/Input';
import { CreateProject } from '@rapidaai/react';
import { CreateProjectResponse } from '@rapidaai/react';
import { useForm } from 'react-hook-form';
import { useCredential } from '@/hooks/use-credential';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { ModalProps } from '@/app/components/base/modal';
import { useRapidaStore } from '@/hooks';
import { ServiceError } from '@rapidaai/react';
import { CenterModal } from '@/app/components/base/modal/content-modal';
import { connectionConfig } from '@/configs';
/**
 *
 * @param props
 * @returns
 */
export const ChangePasswordDialog = (props: ModalProps) => {
  /**
   * form submit
   */
  const { register, handleSubmit } = useForm();

  /**
   * loading context
   */
  const { showLoader, hideLoader } = useRapidaStore();

  /**
   * Credentials
   */
  const [userId, token] = useCredential();
  const [error, setError] = useState<string>();

  const onChangePassword = data => {
    showLoader('overlay');
    CreateProject(
      connectionConfig,
      data.projectName,
      data.projectDescription,
      {
        authorization: token,
        'x-auth-id': userId,
      },
      (err: ServiceError | null, cpr: CreateProjectResponse | null) => {
        hideLoader();
        if (err) {
          setError('unable to process your request. please try again later.');
          return;
        }
        if (cpr?.getSuccess()) {
          props.setModalOpen(false);
        } else {
          setError('Unable to create project please check the details');
        }
      },
    );
  };

  return (
    <CenterModal
      {...props}
      action={'Change password'}
      onSubmit={handleSubmit(onChangePassword)}
    >
      <div>
        <DescriptiveHeading
          heading="Change your password"
          subheading="Add your provider credentials securely and simplify distribution using project credential"
        />

        <fieldset className="space-y-2 col-span-1">
          <Label for="currentPassword" text="Current Password"></Label>
          <Input
            required
            type="text"
            placeholder="**********"
            {...register('currentPassword')}
          ></Input>
        </fieldset>
        <fieldset className="space-y-2 col-span-1">
          <Label for="newPassword" text="New Password"></Label>
          <Input
            required
            type="text"
            placeholder="**********"
            {...register('newPassword')}
          ></Input>
        </fieldset>
        <ErrorMessage message={error} />
      </div>
    </CenterModal>
  );
};
