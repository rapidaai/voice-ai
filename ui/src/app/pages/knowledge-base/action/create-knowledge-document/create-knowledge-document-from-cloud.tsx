import { ConnectKnowledgePage } from '@/app/pages/knowledge-base/action/connect-knowledge';
import { useParams } from 'react-router-dom';
import { FC, useEffect, useState } from 'react';

export const CreateKnowledgeDocumentFromCloudPage: FC<{}> = () => {
  const [knowledgeId, setKnowledgeId] = useState<string | null>(null);
  const { id } = useParams();

  useEffect(() => {
    if (id) {
      setKnowledgeId(id);
    }
  }, [id]);

  if (!knowledgeId) return <div>Please check the url and try again.</div>;
  return <ConnectKnowledgePage knowledgeId={knowledgeId} />;
};
