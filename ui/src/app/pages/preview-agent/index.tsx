import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/Loader/page-loader';

export const PreviewVoiceAgentPage = lazyLoad(
  () => import('./voice-agent/index'),
  module => module.PreviewVoiceAgent,
  {
    fallback: <PageLoader />,
  },
);

export const PreviewPhoneAgentPage = lazyLoad(
  () => import('./voice-agent/index'),
  module => module.PreviewPhoneAgent,
  {
    fallback: <PageLoader />,
  },
);

export const PublicPreviewVoiceAgentPage = lazyLoad(
  () => import('./voice-agent/index'),
  module => module.PublicPreviewVoiceAgent,
  {
    fallback: <PageLoader />,
  },
);
