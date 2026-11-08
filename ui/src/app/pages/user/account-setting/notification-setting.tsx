import { FormLabel } from '@/app/components/ui/primitives';
import { PrimaryButton } from '@/app/components/ui/primitives';
import { ChevronRight } from '@carbon/icons-react';
import { InputCheckbox } from '@/app/components/ui/primitives';
import { FieldSet } from '@/app/components/ui/primitives';
import { InputHelper } from '@/app/components/ui/primitives';
import { RAPIDA_SYSTEM_NOTIFICATION } from '@/models/notification';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { PageActionButtonBlock } from '@/app/components/layout/blocks/page-action-button-block';
import { SectionDivider } from '@/app/components/layout/blocks/section-divider';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { cn } from '@/utils';
import { updateUserNotificationSettings } from '@/clients';

export const NotificationSetting = () => {
  /**
   * loggedin user
   */
  const { token, authId, projectId } = useCurrentCredential();
  /**
   * page error
   */
  const [error, setError] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  /**
   * form handling
   */
  const { register, handleSubmit } = useForm();

  /**
   *
   * @param data
   */
  const onSubmit = (data: any) => {
    setError('');
    setIsSaving(true);
    updateUserNotificationSettings({
      values: data,
      auth: { token, userId: authId, projectId },
    })
      .then(rlp => {
        if (rlp?.getSuccess()) {
          toast.success(
            'The notification setting has been updated successfully.',
          );
        } else {
          let errorMessage = rlp?.getError();
          if (errorMessage) setError(errorMessage.getHumanmessage());
          else {
            setError('Unable to process your request. please try again later.');
          }
          return;
        }
      })
      .catch(() => {
        setError('Unable to process your request. please try again later.');
      })
      .finally(() => {
        setIsSaving(false);
      });
  };

  return (
    <form
      className="flex flex-1 h-full overflow-hidden"
      onSubmit={handleSubmit(onSubmit)} // Use the onSubmit handler
    >
      <div className="flex-1 min-h-0 flex flex-col bg-surface ">
        {/* Scrollable region */}
        <div className="flex-1 min-h-0 overflow-y-auto">
          <header className="px-8 pt-8 pb-6 border-b border-border-subtle ">
            <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-muted mb-1.5">
              Account Settings
            </p>
            <h1 className="text-xl font-semibold text-foreground leading-tight">
              Notification Preferences
            </h1>
            <p className="text-sm text-muted mt-1.5 leading-relaxed">
              Choose which email notifications you want to receive for system
              and workspace events.
            </p>
          </header>

          <div className="px-8 pt-8 pb-10 max-w-5xl flex flex-col gap-8">
            {RAPIDA_SYSTEM_NOTIFICATION.map(notificationCategory => (
              <section
                className="flex flex-col gap-4"
                key={notificationCategory.category}
              >
                <SectionDivider label={notificationCategory.category} />
                <div className="border border-border-subtle bg-surface ">
                  {notificationCategory.items.map((item, idx) => (
                    <div
                      className={cn(
                        'flex items-start gap-3 px-4 py-4',
                        idx !== notificationCategory.items.length - 1 &&
                          'border-b border-border-subtle ',
                      )}
                      key={item.id}
                    >
                      <div className="flex h-5 shrink-0 items-center pt-0.5">
                        <InputCheckbox
                          {...register(item.id)}
                          defaultChecked={item.default}
                        />
                      </div>
                      <FieldSet className="gap-1">
                        <FormLabel htmlFor={item.id} className="text-sm">
                          {item.label}
                        </FormLabel>
                        <InputHelper id={`${item.id}-description`}>
                          {item.description}
                        </InputHelper>
                      </FieldSet>
                    </div>
                  ))}
                </div>
              </section>
            ))}
          </div>
        </div>
        <PageActionButtonBlock errorMessage={error}>
          <div className="flex-1" />
          <div className="h-full flex">
            <PrimaryButton
              size="md"
              renderIcon={ChevronRight}
              type="submit"
              isLoading={isSaving}
              disabled={isSaving}
              className="h-full min-w-[12rem] px-6 rounded-none border-l border-border-subtle "
            >
              Save changes
            </PrimaryButton>
          </div>
        </PageActionButtonBlock>
      </div>
    </form>
  );
};
