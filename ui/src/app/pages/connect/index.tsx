import { lazyLoad } from '@/utils/loadable';
import { LineLoader } from '@/app/components/Loader/line-loader';

export const ConnectGoogleDriveKnowledge = lazyLoad(
  () => import('./connect-knowledge/google-drive'),
  module => module.ConnectGoogleDriveKnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

export const ConnectConfluenceKnowledge = lazyLoad(
  () => import('./connect-knowledge/confluence'),
  module => module.ConnectConfluencePage,
  {
    fallback: <LineLoader />,
  },
);

export const ConnectNotionKnowledge = lazyLoad(
  () => import('./connect-knowledge/notion'),
  module => module.ConnectNotionKnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

export const ConnectSharePointKnowledge = lazyLoad(
  () => import('./connect-knowledge/share-point'),
  module => module.ConnectSharePointKnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

export const ConnectOneDriveKnowledge = lazyLoad(
  () => import('./connect-knowledge/one-drive'),
  module => module.ConnectOneDriveKnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

export const ConnectGithubKnowledge = lazyLoad(
  () => import('./connect-knowledge/github'),
  module => module.ConnectGithubKnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

//
// Action
//
export const ConnectGmailAction = lazyLoad(
  () => import('./connect-action/gmail'),
  module => module.ConnectGmailActionPage,
  {
    fallback: <LineLoader />,
  },
);

export const ConnectGithubAction = lazyLoad(
  () => import('./connect-action/github'),
  module => module.ConnectGithubActionPage,
  {
    fallback: <LineLoader />,
  },
);

export const ConnectGoogleDriveAction = lazyLoad(
  () => import('./connect-action/google-drive'),
  module => module.ConnectGoogleDriveActionPage,
  {
    fallback: <LineLoader />,
  },
);

export const ConnectSlackAction = lazyLoad(
  () => import('./connect-action/slack'),
  module => module.ConnectSlackActionPage,
  {
    fallback: <LineLoader />,
  },
);

// /connect-crm/hubspot
export const ConnectHubspotCRM = lazyLoad(
  () => import('./connect-general/hubspot'),
  module => module.ConnectHubspotCRMPage,
  {
    fallback: <LineLoader />,
  },
);
