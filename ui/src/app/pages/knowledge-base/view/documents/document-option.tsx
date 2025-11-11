import { CardOptionMenu } from '@/app/components/Menu';
import { cn } from '@/styles/media';
import { KnowledgeDocument } from '@rapidaai/react';
import { ReloadIcon } from '@/app/components/Icon/Reload';

/**
 *
 * @param props
 * @returns
 */
export function DocumentOption(props: {
  document: KnowledgeDocument;
  onReloadIndex: (
    knowledgeId: string,
    knowledgeDocumentIds: string[],
    indexType: string,
  ) => void;
}) {
  const options = [
    {
      option: (
        <div className="flex items-center text-sm">
          <span>Re-index the document</span>
          <ReloadIcon className="w-4 h-4 ml-2" />
        </div>
      ),
      onActionClick: () => {
        props.onReloadIndex(
          props.document.getKnowledgeid(),
          [props.document.getId()],
          'paragraph-index',
        );
      },
    },
  ];
  return (
    <>
      <CardOptionMenu classNames={cn('w-9 h-9')} options={options} />
    </>
  );
}
