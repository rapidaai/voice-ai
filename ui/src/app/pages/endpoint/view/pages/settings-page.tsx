import { Notification } from '@/app/components/ui/feedback';
import {
  CopyButton,
  Form,
  InputGroup,
  PrimaryButton,
  Stack,
  TertiaryButton,
  TextArea,
  TextInput,
} from '@/app/components/ui/primitives';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import { useEndpointPageStore } from '@/stores/endpoint';
import { Endpoint } from '@rapidaai/react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Tag,
  Toggletip as CarbonToggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';
import { FormEvent, ReactNode, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';

const Toggletip = (CarbonToggletip as any).default || CarbonToggletip;

function FieldLabelWithToggletip(props: {
  label: string;
  description: ReactNode;
}) {
  return (
    <span className="inline-flex items-center gap-1.5 text-foreground">
      <span>{props.label}</span>
      <Toggletip align="right">
        <ToggletipButton
          label={`${props.label} information`}
          title={`${props.label} information`}
        >
          <Information size={14} />
        </ToggletipButton>
        <ToggletipContent>{props.description}</ToggletipContent>
      </Toggletip>
    </span>
  );
}

export function EndpointSettingsPage(props: { currentEndpoint: Endpoint }) {
  const { currentEndpoint } = props;
  const [userId, token, projectId] = useCredential();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const { onShowEditTagVisible, onUpdateEndpointDetail } =
    useEndpointPageStore();
  const [name, setName] = useState(currentEndpoint.getName());
  const [description, setDescription] = useState(
    currentEndpoint.getDescription() || '',
  );
  const [errorMessage, setErrorMessage] = useState('');
  const endpointId = currentEndpoint.getId();
  const tags = currentEndpoint.getEndpointtag()?.getTagList() ?? [];

  useEffect(() => {
    setName(currentEndpoint.getName());
    setDescription(currentEndpoint.getDescription() || '');
  }, [currentEndpoint]);

  const saveDetails = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!name.trim()) {
      setErrorMessage('Please provide an endpoint name.');
      return;
    }

    setErrorMessage('');
    showLoader('block');
    onUpdateEndpointDetail(
      endpointId,
      name.trim(),
      description.trim(),
      projectId,
      token,
      userId,
      error => {
        hideLoader();
        setErrorMessage(error);
      },
      () => {
        hideLoader();
        toast.success('The endpoint has been successfully updated.');
      },
    );
  };

  return (
    <main className="flex w-full flex-1 flex-col overflow-auto bg-surface ">
      <div className="border-b border-border-subtle px-4 pb-6 pt-4 ">
        <Breadcrumb noTrailingSlash className="mb-2">
          <BreadcrumbItem href={`/deployment/endpoint/${endpointId}/overview`}>
            Endpoint
          </BreadcrumbItem>
        </Breadcrumb>
        <h1 className="text-2xl font-light tracking-tight">General Settings</h1>
      </div>

      <InputGroup title="Identity" childClass="opacity-100!">
        <div className="flex max-w-full flex-col items-start gap-2">
          <FieldLabelWithToggletip
            label="Endpoint ID"
            description="Your endpoint's unique identifier. This cannot be changed."
          />
          <div className="inline-flex max-w-full items-center gap-1">
            <span className="min-w-0 max-w-[32rem] truncate font-mono text-sm text-muted ">
              {endpointId}
            </span>
            <CopyButton className="h-6 w-6 shrink-0">{endpointId}</CopyButton>
          </div>
        </div>
      </InputGroup>

      <InputGroup title="General Information" childClass="opacity-100!">
        <Form className="flex max-w-2xl flex-col gap-8" onSubmit={saveDetails}>
          <Stack gap={6}>
            <TextInput
              id="endpoint-name"
              labelText={
                <FieldLabelWithToggletip
                  label="Name"
                  description="The display name shown across the platform."
                />
              }
              value={name}
              onChange={event => setName(event.target.value)}
              placeholder="e.g. Customer support analysis"
            />
            <TextArea
              id="endpoint-description"
              labelText={
                <FieldLabelWithToggletip
                  label="Description"
                  description="Describe what this endpoint does and its intended use case."
                />
              }
              value={description}
              rows={4}
              onChange={event => setDescription(event.target.value)}
              placeholder="Describe what this endpoint does."
            />
            {errorMessage && (
              <Notification
                kind="error"
                title="Error"
                subtitle={errorMessage}
              />
            )}
            <div>
              <PrimaryButton size="md" type="submit" isLoading={loading}>
                Save changes
              </PrimaryButton>
            </div>
          </Stack>
        </Form>
      </InputGroup>

      <InputGroup title="Tags" childClass="opacity-100!">
        <div className="flex max-w-2xl flex-col items-start gap-4">
          <div className="flex flex-wrap gap-2">
            {tags.length > 0 ? (
              tags.map(tag => (
                <Tag key={tag} size="sm" type="gray">
                  {tag}
                </Tag>
              ))
            ) : (
              <p className="text-sm text-muted">No tags assigned.</p>
            )}
          </div>
          <TertiaryButton
            size="md"
            onClick={() => onShowEditTagVisible(currentEndpoint)}
          >
            Edit tags
          </TertiaryButton>
        </div>
      </InputGroup>
    </main>
  );
}
