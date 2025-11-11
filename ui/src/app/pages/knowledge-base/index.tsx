import { lazyLoad } from '@/utils/loadable';
import { LineLoader } from '@/app/components/Loader/line-loader';

export const KnowledgePage = lazyLoad(
  () => import('./listing'),
  module => module.KnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

// create-knowledge
export const KnowledgeCreateKnowledgePage = lazyLoad(
  () => import('./action/create-knowledge'),
  module => module.CreateKnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

export const KnowledgeConnectKnowledgePage = lazyLoad(
  () => import('./action/connect-knowledge'),
  module => module.ConnectKnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

export const KnowledgeViewKnowledgePage = lazyLoad(
  () => import('./view'),
  module => module.ViewKnowledgePage,
  {
    fallback: <LineLoader />,
  },
);

export const KnowledgeAddNewKnowledgeFilePage = lazyLoad(
  () => import('./action/create-knowledge-document'),
  module => module.CreateKnowledgeDocumentPage,
  {
    fallback: <LineLoader />,
  },
);
export const KnowledgeAddNewKnowledgeCloudFilePage = lazyLoad(
  () => import('./action/create-knowledge-document'),
  module => module.CreateKnowledgeDocumentFromCloudPage,
  {
    fallback: <LineLoader />,
  },
);

export const KnowledgeAddNewStructureDocumentPage = lazyLoad(
  () => import('./action/create-knowledge-document'),
  module => module.CreateKnowledgeStructureDocumentPage,
  {
    fallback: <LineLoader />,
  },
);
// export const KnowledgeDocumentSegmentPage = lazyLoad(
//   () => import('./view/document-segments'),
//   module => module.DocumentSegmentPage,
//   {
//     fallback: <LineLoader />,
//   },
// );
