import { FC } from 'react';
import { Information } from '@carbon/icons-react';
import { Link } from '@carbon/react';
import { useDocumentationUrl } from '@/theme/documentation-url';

export const DocNoticeBlock: FC<{
  children: React.ReactNode;
  docPath?: string;
  docUrl?: string;
  linkText?: string;
}> = ({ children, docPath = '', docUrl, linkText = 'Read documentation' }) => {
  const configuredDocUrl = useDocumentationUrl(docPath);

  return (
    <div className="flex items-center gap-3 w-full px-4 py-3 bg-blue-50 dark:bg-blue-900/20 border-l-[3px] border-blue-600 dark:border-blue-400">
      <Information
        size={20}
        className="shrink-0 text-blue-600 dark:text-blue-400"
      />
      <span className="text-sm flex-1 text-gray-800 dark:text-gray-200">
        {children}
      </span>
      <Link
        href={docUrl ?? configuredDocUrl}
        target="_blank"
        rel="noopener noreferrer"
        className="!font-semibold shrink-0"
      >
        {linkText}
      </Link>
    </div>
  );
};
