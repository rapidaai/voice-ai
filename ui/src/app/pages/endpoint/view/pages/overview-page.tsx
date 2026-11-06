import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { Code, SourceControl } from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  HeaderGlobalAction,
  HeaderGlobalBar,
} from '@carbon/react';
import { useNavigate } from 'react-router-dom';

import { EndpointActivityDashboard } from './endpoint-activity-dashboard';
import { useEndpointPageStore } from '@/stores/endpoint';

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
      <header
        className="flex h-12 shrink-0 items-center justify-between border-b border-border-subtle bg-layer"
        data-testid="endpoint-page-header"
      >
        <div className="min-w-0 pl-4">
          <Breadcrumb noTrailingSlash>
            <BreadcrumbItem href="/deployment/endpoint">
              Endpoints
            </BreadcrumbItem>
            <BreadcrumbItem isCurrentPage>
              <span className="block max-w-[42vw] truncate">
                {currentEndpoint.getName()}
              </span>
            </BreadcrumbItem>
          </Breadcrumb>
        </div>
        <HeaderGlobalBar
          aria-label="Endpoint overview header actions"
          className="h-full items-center"
        >
          <HeaderGlobalAction
            aria-label="View integration"
            tooltipAlignment="end"
            onClick={onShowInstruction}
          >
            <Code size={20} />
          </HeaderGlobalAction>
          <Button
            aria-label="Create new version"
            kind="primary"
            size="lg"
            renderIcon={SourceControl}
            className="h-full! min-h-full! items-center justify-center whitespace-nowrap"
            onClick={() =>
              navigate(
                `/deployment/endpoint/${endpointId}/create-endpoint-version`,
              )
            }
          >
            Create new version
          </Button>
        </HeaderGlobalBar>
      </header>

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
