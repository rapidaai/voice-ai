import { FC } from 'react';
import { cn } from '@/utils';
import { toHumanReadableRelativeTime } from '@/utils/date';
import { KnowledgeDocument } from '@rapidaai/react';
import { useKnowledgeDocumentPageStore } from '@/stores/knowledge/knowledge-document.store';
import { formatFileSize, formatNumber } from '@/utils/format';
import { DocumentSourcePill } from '@/app/components/domain/pills/document-source-pill';
import { DocumentOption } from '@/app/pages/knowledge-base/view/documents/document-option';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import toast from 'react-hot-toast/headless';
import { TableRow } from '@/app/components/ui/table';
import { TableCell } from '@/app/components/ui/table';
import { LabelCell } from '@/app/components/ui/table';
import { Checkmark, Document, WarningAlt } from '@carbon/icons-react';

/**
 *
 */
interface SingleDocumentProps {
  /**
   * current endpoint
   */
  document: KnowledgeDocument;

  /**
   *
   */
  onReload: () => void;
}

/**
 *
 * @param props
 * @returns
 */
export const SingleDocument: FC<SingleDocumentProps> = ({
  document,
  onReload,
}) => {
  const kdAction = useKnowledgeDocumentPageStore();
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const onerror = (err: string) => {
    hideLoader();
    toast.error(err);
    onReload();
  };
  const onsuccess = (e: boolean) => {
    hideLoader();
    onReload();
  };

  const onReloadIndex = (
    knowledgeId: string,
    knowledgeDocumentId: string[],
    indexType: string,
  ) => {
    showLoader();
    kdAction.indexKnowledgeDocument(
      knowledgeId,
      knowledgeDocumentId,
      indexType,
      projectId,
      token,
      userId,
      onerror,
      onsuccess,
    );
  };
  return (
    <TableRow
      data-id={`doc-${document.getId()}`}
      x-knowledge-id={`knowledge-${document.getKnowledgeid()}`}
    >
      {kdAction.visibleColumn('getStatus') && (
        <TableCell>
          <div className="flex items-center gap-2">
            <span
              className={cn(
                'p-1 w-fit block',
                document.getDisplaystatus() === 'available' &&
                  'text-support-success bg-support-success/20',
                document.getDisplaystatus() === 'error' &&
                  'text-support-error bg-support-error/20',
                document.getDisplaystatus() !== 'available' &&
                  document.getDisplaystatus() !== 'error' &&
                  'text-support-warning bg-support-warning/20',
              )}
            >
              {document.getDisplaystatus() === 'error' ? (
                <WarningAlt className="w-5 h-5" />
              ) : (
                <Checkmark className="w-5 h-5" />
              )}
            </span>
            <div>
              <span
                className={cn(
                  'font-medium block text-xs leading-4 capitalize',
                  document.getDisplaystatus() === 'available' &&
                    'text-support-success',
                  document.getDisplaystatus() === 'error' &&
                    'text-support-error',
                  document.getDisplaystatus() !== 'available' &&
                    document.getDisplaystatus() !== 'error' &&
                    'text-support-warning',
                )}
              >
                {document.getDisplaystatus()}
              </span>
              <span className="text-xs text-muted capitalize truncate">
                {document?.getCreateddate() &&
                  toHumanReadableRelativeTime(document?.getCreateddate()!)}
              </span>
            </div>
          </div>
        </TableCell>
      )}
      {kdAction.visibleColumn('getName') && (
        <TableCell>
          <div className="flex items-center gap-2">
            <div className="p-1.5 border border-border-subtle bg-layer shrink-0">
              <Document className="w-5 h-5" />
            </div>
            <div className="flex flex-col min-w-0">
              <span className="text-sm font-semibold line-clamp-1 text-foreground ">
                {document.getName()}
              </span>
              <span className="text-xs text-muted truncate">
                Uploaded{' '}
                {document.getCreateddate() &&
                  toHumanReadableRelativeTime(document.getCreateddate()!)}
              </span>
            </div>
          </div>
        </TableCell>
      )}
      {kdAction.visibleColumn('getDocumenttype') && (
        <LabelCell className="bg-support-info/10 text-support-info truncate">
          {document
            .getDocumentsource()
            ?.getFieldsMap()
            .get('mimeType')
            ?.getStringValue()}
        </LabelCell>
      )}

      {kdAction.visibleColumn('getDocumentSource') && (
        <TableCell>
          <DocumentSourcePill
            type={document
              .getDocumentsource()
              ?.getFieldsMap()
              .get('type')
              ?.getStringValue()}
            source={document
              .getDocumentsource()
              ?.getFieldsMap()
              .get('source')
              ?.getStringValue()}
          />
        </TableCell>
      )}

      {kdAction.visibleColumn('getDocumentsize') && (
        <LabelCell>{formatFileSize(document.getDocumentsize())}</LabelCell>
      )}

      {kdAction.visibleColumn('getRetrievalcount') && (
        <LabelCell>{formatFileSize(document.getRetrievalcount())}</LabelCell>
      )}

      {kdAction.visibleColumn('getTokencount') && (
        <LabelCell>{formatNumber(document.getTokencount())}</LabelCell>
      )}

      {kdAction.visibleColumn('getWordcount') && (
        <LabelCell>{formatNumber(document.getWordcount())}</LabelCell>
      )}
      {kdAction.visibleColumn('getId') && (
        <LabelCell>{`doc_${document.getId()}`}</LabelCell>
      )}
      <TableCell>
        <DocumentOption document={document} onReloadIndex={onReloadIndex} />
      </TableCell>
    </TableRow>
  );
};
