import { MissionBox } from '@/app/components/container/mission-box';
import { ProtectedBox } from '@/app/components/container/protected-box';

import {
  KnowledgeAddNewKnowledgeCloudFilePage,
  KnowledgeAddNewKnowledgeFilePage,
  KnowledgeCreateKnowledgePage,
  KnowledgeViewKnowledgePage,
  KnowledgeAddNewStructureDocumentPage,
  KnowledgePage,
} from '@/app/pages/knowledge-base';
import { Outlet, Route, Routes } from 'react-router-dom';

export const KnowledgeRoute = () => {
  return (
    <Routes>
      <Route
        path=""
        element={
          <ProtectedBox>
            <MissionBox>
              <Outlet />
            </MissionBox>
          </ProtectedBox>
        }
      >
        <Route index element={<KnowledgePage />} />
        <Route
          path={'create-knowledge'}
          element={<KnowledgeCreateKnowledgePage />}
        />

        <Route path={':id'} element={<KnowledgeViewKnowledgePage />} />

        <Route
          path={':id/add-knowledge-file'}
          element={<KnowledgeAddNewKnowledgeFilePage />}
        />

        <Route
          path={':id/add-cloud-file'}
          element={<KnowledgeAddNewKnowledgeCloudFilePage />}
        />

        <Route
          path={':id/add-structure-file'}
          element={<KnowledgeAddNewStructureDocumentPage />}
        />

        {/* <Route
          path={':id/:knowledgeDocumentId/document-segments'}
          element={<KnowledgeDocumentSegmentPage />}
        /> */}
      </Route>
    </Routes>
  );
};
