/**
 * Asynchronously loads the component for HomePage
 */

import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const StaticPrivacyPage = lazyLoad(
  () => import('./privacy'),
  module => module.PrivacyPage,
  {
    fallback: <PageLoader />,
  },
);

export const StaticTermsPage = lazyLoad(
  () => import('./terms'),
  module => module.TermsPage,
  {
    fallback: <PageLoader />,
  },
);

export const StaticPageNotFoundPage = lazyLoad(
  () => import('./404'),
  module => module.PageNotFoundPage,
  {
    fallback: <PageLoader />,
  },
);
