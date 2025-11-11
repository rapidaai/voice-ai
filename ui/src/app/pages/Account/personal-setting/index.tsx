import React, { useCallback, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { Label } from '@/app/components/Form/Label';
import { Input } from '@/app/components/Form/Input';
import { ArrowButton } from '@/app/components/Form/Button/ArrowButton';
import { UpdateUser, GetUser } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { GetUserResponse, UpdateUserResponse } from '@rapidaai/react';
import { ChangePasswordDialog } from '@/app/components/base/modal/change-password-modal';
import { useForm } from 'react-hook-form';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import { DescriptiveHeading } from '@/app/components/Heading/DescriptiveHeading';
import { BorderButton } from '@/app/components/Form/Button';
import { connectionConfig } from '@/configs';
export function PersonalSettingPage() {
  /**
   *
   */
  const [changePasswordModalOpen, setChangePasswordModalOpen] = useState(false);
  const [userDetails, setUserDetails] = useState<{
    name: string;
    email: string;
  }>({ name: '', email: '' });
  /**
   *
   */
  const { register, handleSubmit } = useForm();
  const [error, setError] = useState<string>();
  /**
   * Credentials
   */
  const [userId, token] = useCredential();

  /**
   * Loading context
   */

  const { showLoader, hideLoader } = useRapidaStore();

  const afterUpdateUser = useCallback(
    (err: ServiceError | null, auth: UpdateUserResponse | null) => {
      hideLoader();
      if (err) {
        toast.error('Unable to process your request. please try again later.');
        return;
      }
      if (auth?.getSuccess()) {
        toast.success('Your profile has been updated successfully.');
      } else {
        toast.error('Unable to process your request. please try again later.');
        return;
      }
    },
    [],
  );
  /**
   *
   */
  const onSaveUserProfile = data => {
    if (data?.userName.trim() === '') {
      setError('cannot set an empty username');
      return;
    }
    showLoader();
    UpdateUser(
      connectionConfig,
      afterUpdateUser,
      {
        authorization: token,
        'x-auth-id': userId,
      },
      data.userName,
    );
  };

  useEffect(() => {
    GetUser(
      connectionConfig,
      (err: ServiceError | null, gur: GetUserResponse | null) => {
        if (err) {
          return;
        }
        if (gur?.getSuccess()) {
          setUserDetails({
            email: gur.getData()?.getEmail() as string,
            name: gur.getData()?.getName() as string,
          });

          gur.getData()?.getName();
        }
      },
      {
        authorization: token,
        'x-auth-id': userId,
      },
    );
  }, []);

  /**
   *
   */
  return (
    <>
      <Helmet title="Personal Settings"></Helmet>
      <div className="flex items-center justify-between py-2 px-4">
        <DescriptiveHeading
          heading="Personal Settings"
          subheading="Take a moment to update your profile."
        />
      </div>
      <form
        className="space-y-8 px-4 border-t dark:border-gray-800"
        onSubmit={handleSubmit(onSaveUserProfile)}
      >
        <div className="items-center md:mx-0 m-5 grid grid-cols-9 gap-4 w-full">
          <div className="space-y-2 col-span-3">
            <Label for="userName" text="Name"></Label>
            <Input
              value={userDetails?.name ?? ''}
              type="text"
              required
              placeholder="eg: John deo"
              {...register('userName')}
              onChange={e => {
                setError('');
                setUserDetails({
                  name: e.target.value,
                  email: userDetails.email,
                });
              }}
            ></Input>
          </div>
          <div className="disabled space-y-2 col-span-3 bg-y-gradient-white-grey-200 overflow-hidden">
            <Label for="userEmail" text="Email"></Label>
            <Input
              disabled={true}
              value={userDetails?.email ?? ''}
              type="email"
              placeholder="eg: john@rapida.ai"
              {...register('userEmail')}
            ></Input>
          </div>
        </div>

        <section className="hidden">
          <p className="text-sm">
            No longer want to use our service? You can delete your account here.
            This action is not reversible. All information related to this
            account will be deleted permanently.
          </p>

          <div className="md:mx-0 m-5 flex space-y-2 w-full flex-col">
            <div className="w-60">
              <BorderButton type="button" onClick={() => {}}>
                Delete Account
              </BorderButton>
            </div>
          </div>
        </section>
        <footer className="border-t pt-8 dark:border-gray-700">
          {error && (
            <fieldset>
              <p className="p-4 text-sm text-red-800 rounded-lg bg-red-50 dark:bg-gray-900/50 capitalize dark:text-red-400 font-semibold">
                {error}
              </p>
            </fieldset>
          )}
          <ul className="flex justify-end">
            {/* <li className="ml-0 opacity-80">
                <BorderedButton type="button" onClick={() => {}} size="sm">
                  <span className="text-sm">Cancel</span>
                </BorderedButton>
              </li> */}
            <li className="">
              <ArrowButton type="submit" label="Save Changes"></ArrowButton>
            </li>
          </ul>
        </footer>
      </form>
      <ChangePasswordDialog
        setModalOpen={setChangePasswordModalOpen}
        modalOpen={changePasswordModalOpen}
      />
    </>
  );
}
