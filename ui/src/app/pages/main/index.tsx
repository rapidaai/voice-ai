import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const DashboardHomePage = lazyLoad(
  () => import('./web-dashboard'),
  module => module.HomePage,
  {
    fallback: <PageLoader />,
  },
);
