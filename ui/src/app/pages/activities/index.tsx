import { lazyLoad } from '@/utils/loadable';
import { LineLoader } from '@/app/components/Loader/line-loader';

export const LLMActivityListingPage = lazyLoad(
  () => import('./llm-activities'),
  module => module.ListingPage,
  {
    fallback: <LineLoader />,
  },
);

export const WebhookActivityListingPage = lazyLoad(
  () => import('./webhook-activities'),
  module => module.ListingPage,
  {
    fallback: <LineLoader />,
  },
);

export const ConversationActivityListingPage = lazyLoad(
  () => import('./conversation-activities'),
  module => module.ListingPage,
  {
    fallback: <LineLoader />,
  },
);

export const KnowledgeActivityListingPage = lazyLoad(
  () => import('./knowledge-activities'),
  module => module.ListingPage,
  {
    fallback: <LineLoader />,
  },
);

export const ToolActivityListingPage = lazyLoad(
  () => import('./tool-activities'),
  module => module.ListingPage,
  {
    fallback: <LineLoader />,
  },
);
