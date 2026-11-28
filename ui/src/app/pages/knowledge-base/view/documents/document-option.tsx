import { CardOptionMenu } from '@/app/components/ui/composites';
import { cn } from '@/utils';
import { KnowledgeDocument } from '@rapidaai/react';
import { Renew } from '@carbon/icons-react';

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
      label: 'Re-index the document',
      renderIcon: Renew,
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
      <CardOptionMenu
        classNames={cn(
          '[&_.cds--overflow-menu]:h-9 [&_.cds--overflow-menu]:w-9',
        )}
        options={options}
      />
    </>
  );
}
