import { lazyLoad } from '@/utils/loadable';
import { PageLoader } from '@/app/components/ui/feedback';

export const PreviewVoiceAgentPage = lazyLoad(
  () => import('./voice-agent/index'),
  module => module.PreviewVoiceAgent,
  {
    fallback: <PageLoader />,
  },
);

export const PreviewVoiceAgentV2Page = lazyLoad(
  () => import('./voice-agent/v2'),
  module => module.PreviewVoiceAgentV2,
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
