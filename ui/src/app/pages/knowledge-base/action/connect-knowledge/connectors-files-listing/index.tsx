import { Content } from '@rapidaai/react';
import { ToolProvider } from '@rapidaai/react';
import { ConfluenceKnowledgeFileListing } from '@/app/pages/knowledge-base/action/connect-knowledge/connectors-files-listing/confluence';
import { GithubKnowledgeFileListing } from '@/app/pages/knowledge-base/action/connect-knowledge/connectors-files-listing/github';
import { GitlabKnowledgeFileListing } from '@/app/pages/knowledge-base/action/connect-knowledge/connectors-files-listing/gitlab';
import { GoogleDriveKnowledgeFileListing } from '@/app/pages/knowledge-base/action/connect-knowledge/connectors-files-listing/google-drive';
import { NotionKnowledgeFileListing } from '@/app/pages/knowledge-base/action/connect-knowledge/connectors-files-listing/notion';
import { OneDriveKnowledgeFileListing } from '@/app/pages/knowledge-base/action/connect-knowledge/connectors-files-listing/one-drive';
import { SharePointKnowledgeFileListing } from '@/app/pages/knowledge-base/action/connect-knowledge/connectors-files-listing/share-point';
import { ConnectorFileContext } from '@/hooks/use-connector-file-page-store';
import { FC, HTMLAttributes, useContext, useEffect } from 'react';

export interface KnowledgeFileListingProps
  extends HTMLAttributes<HTMLDivElement> {
  /**
   * tools for that the listing and configuration will happen
   */
  toolProvider: ToolProvider;

  /**
   *
   * @returns
   */
  onChangeContents: (cnts: Array<Content>) => void;
}

export const KnowledgeFileListing: FC<KnowledgeFileListingProps> = props => {
  const ctx = useContext(ConnectorFileContext);

  useEffect(() => {
    ctx.clear();
  }, [JSON.stringify(props.toolProvider)]);

  //
  if (props.toolProvider.getName() === 'Microsoft OneDrive')
    return <OneDriveKnowledgeFileListing {...props} />;
  if (props.toolProvider.getName() === 'Confluence')
    return <ConfluenceKnowledgeFileListing {...props} />;
  if (props.toolProvider.getName() === 'GitHub')
    return <GithubKnowledgeFileListing {...props} />;
  if (props.toolProvider.getName() === 'Gitlab')
    return <GitlabKnowledgeFileListing {...props} />;
  if (props.toolProvider.getName() === 'Notion')
    return <NotionKnowledgeFileListing {...props} />;
  if (props.toolProvider.getName() === 'SharePoint')
    return <SharePointKnowledgeFileListing {...props} />;
  return <GoogleDriveKnowledgeFileListing {...props} />;
};
