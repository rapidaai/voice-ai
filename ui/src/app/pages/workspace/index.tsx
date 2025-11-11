import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const OrganizationAccessSecurityPage = lazyLoad(
  () => import('./access-security'),
  module => module.AccessSecurityPage,
  {
    fallback: <PageLoader />,
  },
);

export const OrganizationOverviewPage = lazyLoad(
  () => import('./overview'),
  module => module.OverviewPage,
  {
    fallback: <PageLoader />,
  },
);

export const OrganizationBillingPage = lazyLoad(
  () => import('./billing'),
  module => module.BillingPage,
  {
    fallback: <PageLoader />,
  },
);

export const OrganizationUserPage = lazyLoad(
  () => import('./user'),
  module => module.UserPage,
  {
    fallback: <PageLoader />,
  },
);

export const OrganizationProjectPage = lazyLoad(
  () => import('./project'),
  module => module.ProjectPage,
  {
    fallback: <PageLoader />,
  },
);
