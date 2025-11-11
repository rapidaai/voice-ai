import {
  AssistantDefinition,
  ConnectionConfig,
  DeleteAssistant,
  GetAssistant,
  GetAssistantRequest,
} from '@rapidaai/react';
import { GetAssistantResponse } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { ErrorContainer } from '@/app/components/error-container';
import { FormLabel } from '@/app/components/form-label';
import {
  IBlueBGButton,
  ICancelButton,
  IRedBorderButton,
} from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { CopyInput } from '@/app/components/Form/Input/copy-input';
import { Textarea } from '@/app/components/Form/Textarea';
import { InputGroup } from '@/app/components/input-group';
import { InputHelper } from '@/app/components/input-helper';
import { useDeleteConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-delete-confirmation';
import { useRapidaStore } from '@/hooks';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import { UpdateAssistantDetail } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { RedNoticeBlock } from '@/app/components/container/message/notice-block';

export function EditAssistantPage() {
  /**
   * get all the models when type change
   */
  const { assistantId } = useParams();
  const { goToAssistantListing } = useGlobalNavigation();

  if (!assistantId)
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={goToAssistantListing}
          code="403"
          actionLabel="Go to listing"
          title="Assistant not available"
          description="This assistant may be archived or you don't have access to it. Please check with your administrator or try another assistant."
        />
      </div>
    );

  return <EditAssistant assistantId={assistantId!} />;
}
export const EditAssistant: FC<{ assistantId: string }> = ({ assistantId }) => {
  const { authId, token, projectId } = useCurrentCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const { goToAssistantListing } = useGlobalNavigation();

  useEffect(() => {
    showLoader('block');

    const request = new GetAssistantRequest();
    const assistantDef = new AssistantDefinition();
    assistantDef.setAssistantid(assistantId);
    request.setAssistantdefinition(assistantDef);
    GetAssistant(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    )
      .then(car => {
        hideLoader();
        if (car?.getSuccess()) {
          const assistant = car.getData();
          if (assistant) {
            setName(assistant.getName());
            setDescription(assistant.getDescription());
          }
        } else {
          const error = car?.getError();
          if (error) {
            toast.error(error.getHumanmessage());
            return;
          }
          toast.error('Unable to delete assistant. please try again later.');
          return;
        }
      })
      .catch(err => {
        hideLoader();
      });
  }, [assistantId]);

  const onUpdateAssistantDetail = () => {
    showLoader('block');
    const afterUpdateAssistant = (
      err: ServiceError | null,
      car: GetAssistantResponse | null,
    ) => {
      hideLoader();
      if (car?.getSuccess()) {
        toast.success('The assistant has been successfully updated.');
        const assistant = car.getData();
        if (assistant) {
          setName(assistant.getName());
          setDescription(assistant.getDescription());
        }
      } else {
        const error = car?.getError();
        if (error) {
          setErrorMessage(error.getHumanmessage());
          return;
        }
        setErrorMessage('Unable to update assistant. please try again later.');
        return;
      }
    };
    UpdateAssistantDetail(
      connectionConfig,
      assistantId,
      name,
      description,
      afterUpdateAssistant,
      {
        authorization: token,
        'x-auth-id': authId,
        'x-project-id': projectId,
      },
    );
  };

  // call it when you want to delete the assistant
  const Deletion = useDeleteConfirmDialog({
    onConfirm: () => {
      showLoader('block');
      const afterDeleteAssistant = (
        err: ServiceError | null,
        car: GetAssistantResponse | null,
      ) => {
        if (car?.getSuccess()) {
          toast.error('The assistant has been deleted successfully.');
          goToAssistantListing();
        } else {
          const error = car?.getError();
          if (error) {
            toast.error(error.getHumanmessage());
            return;
          }
          toast.error('Unable to delete assistant. please try again later.');
          return;
        }
      };

      DeleteAssistant(connectionConfig, assistantId, afterDeleteAssistant, {
        authorization: token,
        'x-auth-id': authId,
        'x-project-id': projectId,
      });
    },
    name: name,
  });

  //
  return (
    <div className="w-full flex flex-col flex-1">
      <Deletion.ConfirmDeleteDialogComponent />
      <PageHeaderBlock className="border-b">
        <PageTitleBlock>General Settings</PageTitleBlock>
      </PageHeaderBlock>
      <div className="overflow-auto flex flex-col flex-1 pb-20 bg-white dark:bg-gray-900">
        <div className="p-5 space-y-6">
          <div className="grid grid-cols-3 gap-6">
            <FieldSet>
              <FormLabel>ID</FormLabel>
              <CopyInput
                name="id"
                disabled
                value={assistantId}
                className="cursor-copy bg-light-background"
                placeholder="eg: your emotion detector"
              ></CopyInput>
            </FieldSet>
            <FieldSet>
              <FormLabel>Name</FormLabel>
              <Input
                name="usecase"
                className="bg-light-background"
                onChange={e => {
                  setName(e.target.value);
                }}
                value={name}
                placeholder="eg: your emotion detector"
              ></Input>
            </FieldSet>
            <FieldSet className="col-span-2">
              <FormLabel>Description</FormLabel>
              <Textarea
                row={5}
                className="bg-light-background"
                value={description}
                placeholder={"What's the purpose of the assistant?"}
                onChange={t => setDescription(t.target.value)}
              />
            </FieldSet>
          </div>
        </div>
        <InputGroup title="Permanent Actions" initiallyExpanded={false}>
          <RedNoticeBlock>
            Active connections will be terminated immediately, and the data will
            be permanently deleted after the rolling period.
          </RedNoticeBlock>
          <div className="flex flex-row items-center justify-between p-6">
            <FieldSet>
              <p className="font-semibold">Delete this assistant</p>
              <InputHelper className="-mt-2">
                Once you delete a assistant, there is no going back. Please be
                certain
              </InputHelper>
            </FieldSet>
            <IRedBorderButton
              className="rounded-[2px]"
              isLoading={loading}
              onClick={Deletion.showDialog}
            >
              Delete
            </IRedBorderButton>
          </div>
        </InputGroup>
      </div>
      <PageActionButtonBlock errorMessage={errorMessage}>
        <ICancelButton
          className="px-4 rounded-[2px]"
          onClick={() => {
            // showDialog(navigator.goBack);
          }}
        >
          Cancel
        </ICancelButton>
        <IBlueBGButton
          type="button"
          isLoading={loading}
          onClick={onUpdateAssistantDetail}
          className="px-4 rounded-[2px]"
        >
          Update assistant
        </IBlueBGButton>
      </PageActionButtonBlock>
    </div>
  );
};
