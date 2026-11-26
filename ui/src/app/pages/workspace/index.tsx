import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/ui/feedback';

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

export const OrganizationRoleCatalogPage = lazyLoad(
  () => import('./role'),
  module => module.RoleCatalogPage,
  { fallback: <PageLoader /> },
);

export const OrganizationRoleDetailPage = lazyLoad(
  () => import('./role/detail'),
  module => module.RoleDetailPage,
  { fallback: <PageLoader /> },
);
