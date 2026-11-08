import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { Code, SourceControl } from '@carbon/icons-react';
import { HeaderGlobalAction } from '@carbon/react';
import { useNavigate } from 'react-router-dom';

import { EndpointActivityDashboard } from './endpoint-activity-dashboard';
import { useEndpointPageStore } from '@/stores/endpoint';
import { ResourceDetailHeader } from '@/app/components/layout/resource-detail-header';

export function EndpointOverviewPage(props: {
  currentEndpoint: Endpoint;
  currentEndpointProviderModel: EndpointProviderModel;
}) {
  const { currentEndpoint } = props;
  const navigate = useNavigate();
  const { onShowInstruction } = useEndpointPageStore();
  const endpointId = currentEndpoint.getId();

  return (
    <main className="flex min-h-0 flex-1 flex-col bg-surface">
      <ResourceDetailHeader
        collectionHref="/deployment/endpoint"
        collectionLabel="Endpoints"
        resourceName={currentEndpoint.getName()}
        actionsLabel="Endpoint overview header actions"
        testId="endpoint-page-header"
        secondaryActions={
          <HeaderGlobalAction
            aria-label="View integration"
            tooltipAlignment="end"
            onClick={onShowInstruction}
          >
            <Code size={20} />
          </HeaderGlobalAction>
        }
        primaryAction={{
          label: 'Create new version',
          icon: SourceControl,
          onClick: () =>
            navigate(
              `/deployment/endpoint/${endpointId}/create-endpoint-version`,
            ),
        }}
      />

      <div className="min-h-0 flex-1 overflow-auto">
        <EndpointActivityDashboard
          endpointId={endpointId}
          onOpenLogs={() => navigate(`/deployment/endpoint/${endpointId}/logs`)}
          onOpenPlayground={() =>
            navigate(`/deployment/endpoint/${endpointId}/playground`)
          }
        />
      </div>
    </main>
  );
}
