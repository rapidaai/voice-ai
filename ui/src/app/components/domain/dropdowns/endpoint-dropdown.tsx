import type { Endpoint } from '@rapidaai/react';
import { useEndpointPageStore } from '@/stores/endpoint';
import { useCredential } from '@/hooks/use-credential';
import { Launch, Renew } from '@carbon/icons-react';
import { useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Dropdown, Button } from '@carbon/react';
import { cn } from '@/utils';

interface EndpointDropdownProps {
  className?: string;
  currentEndpoint?: string;
  onChangeEndpoint: (endpoint: Endpoint) => void;
}

interface EndpointDropdownViewProps extends EndpointDropdownProps {
  endpoints: Endpoint[];
  isLoading?: boolean;
  onCreateEndpoint: () => void;
  onRefresh: () => void;
}

export function EndpointDropdownView({
  className,
  currentEndpoint,
  endpoints,
  isLoading = false,
  onChangeEndpoint,
  onCreateEndpoint,
  onRefresh,
}: EndpointDropdownViewProps) {
  const selectedItem =
    endpoints.find(endpoint => endpoint.getId() === currentEndpoint) || null;

  return (
    <div className={cn(className)}>
      <div className="cds--label domain-connected-dropdown-label">Endpoint</div>
      <div className="domain-connected-dropdown-row flex w-full items-stretch border-b border-b-border-strong bg-field">
        <div className="min-w-0 flex-1">
          <Dropdown
            id="endpoint-dropdown"
            titleText="Endpoint"
            hideLabel
            label="Select endpoint"
            items={endpoints}
            selectedItem={selectedItem}
            disabled={isLoading}
            itemToString={(item: Endpoint | null) =>
              item ? `${item.getName()} [${item.getId()}]` : ''
            }
            onChange={({ selectedItem }) => {
              if (selectedItem) {
                onChangeEndpoint(selectedItem);
              }
            }}
          />
        </div>
        <Button
          hasIconOnly
          renderIcon={Renew}
          iconDescription="Refresh endpoints"
          kind="ghost"
          size="md"
          disabled={isLoading}
          onClick={onRefresh}
          className="domain-connected-dropdown-action shrink-0"
        />
        <Button
          hasIconOnly
          renderIcon={Launch}
          iconDescription="Create endpoint"
          kind="ghost"
          size="md"
          onClick={onCreateEndpoint}
          className="domain-connected-dropdown-action shrink-0"
        />
      </div>
    </div>
  );
}

export function EndpointDropdown(props: EndpointDropdownProps) {
  const [userId, token, projectId] = useCredential();
  const endpointActions = useEndpointPageStore();
  const [isLoading, setIsLoading] = useState(false);

  const showLoader = () => setIsLoading(true);
  const hideLoader = () => setIsLoading(false);

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((data: Endpoint[]) => {
    hideLoader();
  }, []);

  const getEndpoints = useCallback((projectId, token, userId) => {
    showLoader();
    endpointActions.onGetAllEndpoint(
      projectId,
      token,
      userId,
      onError,
      onSuccess,
    );
  }, []);

  useEffect(() => {
    if (props.currentEndpoint) {
      endpointActions.addCriteria('id', props.currentEndpoint, 'or');
    }
    getEndpoints(projectId, token, userId);
  }, [
    projectId,
    endpointActions.page,
    endpointActions.pageSize,
    JSON.stringify(endpointActions.criteria),
    props.currentEndpoint,
  ]);

  return (
    <EndpointDropdownView
      className={props.className}
      currentEndpoint={props.currentEndpoint}
      endpoints={endpointActions.endpoints}
      isLoading={isLoading}
      onChangeEndpoint={props.onChangeEndpoint}
      onRefresh={() => getEndpoints(projectId, token, userId)}
      onCreateEndpoint={() =>
        window.open('/deployment/endpoint/create-endpoint', '_blank')
      }
    />
  );
}
