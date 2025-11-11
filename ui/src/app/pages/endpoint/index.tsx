import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const DeploymentEndpointPage = lazyLoad(
  () => import('./listing'),
  module => module.EndpointPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentViewEndpointPage = lazyLoad(
  () => import('./view'),
  module => module.ViewEndpointPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateEndpointPage = lazyLoad(
  () => import('./actions/create-endpoint'),
  module => module.CreateEndpointPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentConfigureEndpointPage = lazyLoad(
  () => import('./actions/configure-endpoint'),
  module => module.ConfigureEndpointPage,
  {
    fallback: <PageLoader />,
  },
);

export const DeploymentCreateVersionEndpointPage = lazyLoad(
  () => import('./actions/create-endpoint-version'),
  module => module.CreateNewVersionEndpointPage,
  {
    fallback: <PageLoader />,
  },
);
