import { Helmet } from '@/app/components/Helmet';
import { FC, useCallback, useState } from 'react';
import { useCredential, useRapidaStore } from '@/hooks';
import {
  Button,
  IBlueBGButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { KnowledgeDocument } from '@rapidaai/react';
import { TabForm } from '@/app/components/Form/tab-form';
import { useCreateKnowledgeDocumentPageStore } from '@/hooks/use-create-knowledge-document-page-store';
import { ToolProviderContextProvider } from '@/context/tool-provider-context';
import { useToolProviderPageStore } from '@/hooks/use-tool-provider-page-store';
import { ToolProvider } from '@rapidaai/react';
import { SelectToolProvider } from '@/app/components/configuration/tool-provider-config/select-tool-provider';
import { ConnectorFileContextProvider } from '@/context/connector-file-context';
import { useConnectorFilePageStore } from '@/hooks/use-connector-file-page-store';
import { KnowledgeFileListing } from '@/app/pages/knowledge-base/action/connect-knowledge/connectors-files-listing';
import { Content } from '@rapidaai/react';
import { RapidaDocumentSource } from '@/utils/rapida_document';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { MoveRight } from 'lucide-react';

export const ConnectKnowledgePage: FC<{ knowledgeId: string }> = ({
  knowledgeId,
}) => {
  const [activeTab, setActiveTab] = useState('create-knowledge');
  const [errorMessage, setErrorMessage] = useState('');
  const knowledgeDocumentAction = useCreateKnowledgeDocumentPageStore();
  const { goToKnowledge } = useGlobalNavigation();

  const toolAction = useToolProviderPageStore();
  /**
   * all the credentials you will need do things
   */
  const [userId, token, projectId, organizationId] = useCredential();

  /**
   * show and hide loaders
   */
  const { showLoader, hideLoader } = useRapidaStore();

  const onsuccess = useCallback((d: KnowledgeDocument[]) => {
    hideLoader();
    goToKnowledge(knowledgeId);
  }, []);

  /**
   *
   */
  const onerror = useCallback((e: string) => {
    hideLoader();
    setErrorMessage(e);
  }, []);

  const [selectedToolProvider, setSelectToolProvider] =
    useState<ToolProvider | null>(null);
  const connectorFilesActions = useConnectorFilePageStore();

  //   maintain the set of content that will chang e in the case what we need
  const [contents, setContents] = useState<Array<Content>>([]);

  const onChooseToolProviders = tool => {
    if (tool) {
      // reset the content when tool changes
      setContents([]);
      setSelectToolProvider(tool);
    }
  };

  const onChangeContents = (cnts: Array<Content>) => {
    setContents(cnts);
  };

  /**
   *
   */
  const createKnowledgeDocument = () => {
    if (!selectedToolProvider) {
      setErrorMessage('Please provide a valid document source.');
      return;
    }

    if (contents.length === 0) {
      setErrorMessage(
        'Please select one or more documents for connecting to knowledge.',
      );
      return;
    }

    showLoader('overlay');
    knowledgeDocumentAction.onCreateDocument(
      knowledgeId,
      RapidaDocumentSource.TOOL,
      selectedToolProvider.getId(),
      contents,
      projectId,
      token,
      userId,
      onsuccess,
      onerror,
    );
  };

  return (
    <>
      <Helmet title="Connect existing knowledge"></Helmet>
      <TabForm
        activeTab={activeTab}
        onChangeActiveTab={(tabName: string) => {}}
        errorMessage={errorMessage}
        form={[
          {
            name: 'Connect Application',
            description:
              'By creating a knowledge you combine multiple documents from different source.',
            code: 'create-knowledge',
            body: (
              <ToolProviderContextProvider contextValue={toolAction}>
                <SelectToolProvider
                  toolFeature="data.knowledge"
                  selectedToolProvider={selectedToolProvider}
                  onSelectToolProvider={tls => {
                    onChooseToolProviders(tls);
                  }}
                  toolbarClassName="border-b"
                  connectParam={{
                    linker: 'organization',
                    linkerId: organizationId,
                    redirectTo: `/knowledge/${knowledgeId}/connect-knowledge`,
                  }}
                />
              </ToolProviderContextProvider>
            ),
            actions: [
              <ICancelButton
                className="px-8 h-14 items-start"
                onClick={() => {
                  goToKnowledge(knowledgeId);
                }}
              >
                Cancle
              </ICancelButton>,
              <IBlueBGButton
                type="button"
                className="px-8 h-14 text-base font-normal items-start"
                onClick={() => {
                  if (!selectedToolProvider) {
                    setErrorMessage(
                      'Please select one or more data sources to choose the files.',
                    );
                    return;
                  }
                  setErrorMessage('');
                  setActiveTab('upload-document');
                }}
              >
                Connect Knowledge Source
                <MoveRight className="ml-3" strokeWidth={1.5} />
              </IBlueBGButton>,
            ],
          },
          {
            name: 'Add new document',
            description: 'Adding more documents to your knowledge base',
            code: 'upload-document',
            body:
              selectedToolProvider && activeTab === 'upload-document' ? (
                <ConnectorFileContextProvider
                  contextValue={connectorFilesActions}
                >
                  <KnowledgeFileListing
                    toolProvider={selectedToolProvider}
                    className="overflow-auto max-h-[70dvh]"
                    onChangeContents={onChangeContents}
                  />
                </ConnectorFileContextProvider>
              ) : (
                <></>
              ),
            actions: [
              <Button type="button" onClick={createKnowledgeDocument}>
                Upload new document
              </Button>,
            ],
          },
        ]}
      />
    </>
  );
};
