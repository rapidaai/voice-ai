import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const OnboardingCreateOrganizationPage = lazyLoad(
  () => import('./user-organization'),
  module => module.CreateOrganizationPage,
  {
    fallback: <PageLoader />,
  },
);

export const OnboardingCreateProjectPage = lazyLoad(
  () => import('./user-project'),
  module => module.CreateProjectPage,
  {
    fallback: <PageLoader />,
  },
);
