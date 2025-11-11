import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const SplashPage = lazyLoad(
  () => import('./splash-animation'),
  module => module.SplashAnimationPage,
  {
    fallback: <PageLoader />,
  },
);
