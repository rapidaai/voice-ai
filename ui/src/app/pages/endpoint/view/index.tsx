import { EndpointDetailTabContent } from '@/app/pages/endpoint/view/pages';
import { useEndpointPageStore } from '@/stores/endpoint';
import { useOutletContext, useParams } from 'react-router-dom';

type EndpointViewOutletContext = {
  onReload: () => void;
};

export function ViewEndpointPage({ tab: selectedTab }: { tab?: string }) {
  const { currentEndpoint, currentEndpointProviderModel } =
    useEndpointPageStore();

  const { tab: routeTab = 'overview' } = useParams();
  const tab = selectedTab ?? routeTab;
  const { onReload } = useOutletContext<EndpointViewOutletContext>();

  return (
    <EndpointDetailTabContent
      activeTab={tab}
      currentEndpoint={currentEndpoint}
      currentEndpointProviderModel={currentEndpointProviderModel}
      onReload={onReload}
    />
  );
}
