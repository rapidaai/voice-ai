import {
  KnowledgeAddNewKnowledgeFilePage,
  KnowledgeCreateKnowledgePage,
  KnowledgeViewKnowledgePage,
  KnowledgeAddNewStructureDocumentPage,
  KnowledgePage,
} from '@/app/pages/knowledge-base';
import { SkeletonText } from '@carbon/react';
import type { RouteObject } from 'react-router-dom';
import { useKnowledgePageStore } from '@/stores/knowledge/knowledge.store';
import { breadcrumb, type BreadcrumbLabelProps } from './breadcrumb-handle';
import { WorkspaceRouteLayout } from './workspace-route-layout';

function KnowledgeBreadcrumbLabel({ params }: BreadcrumbLabelProps) {
  const knowledge = useKnowledgePageStore(state => state.currentKnowledge);
  if (!knowledge || knowledge.getId() !== params.id) {
    return <SkeletonText className="mb-0!" width="8rem" />;
  }

  return knowledge.getName();
}

export const knowledgeRoute: RouteObject = {
  path: '/knowledge',
  element: <WorkspaceRouteLayout />,
  handle: breadcrumb('Knowledge'),
  children: [
    { index: true, element: <KnowledgePage /> },
    {
      path: 'create-knowledge',
      element: <KnowledgeCreateKnowledgePage />,
      handle: breadcrumb('Create knowledge'),
    },
    {
      path: ':id',
      handle: { breadcrumb: { render: KnowledgeBreadcrumbLabel } },
      children: [
        { index: true, element: <KnowledgeViewKnowledgePage /> },
        {
          path: 'add-knowledge-file',
          element: <KnowledgeAddNewKnowledgeFilePage />,
          handle: breadcrumb('Add file'),
        },
        {
          path: 'add-structure-file',
          element: <KnowledgeAddNewStructureDocumentPage />,
          handle: breadcrumb('Add structured file'),
        },
      ],
    },
  ],
};
