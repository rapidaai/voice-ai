import { ValidHttpUrl } from '@/utils';
import { FileIcon, defaultStyles } from 'react-file-icon';
import { FC } from 'react';
import { cn } from '@/styles/media';

export const FileExtensionIcon: FC<{
  filename: string;
  className?: string;
}> = ({ filename, className }) => {
  if (!filename) return <></>;
  if (ValidHttpUrl(filename))
    return (
      <div
        className={cn('w-5 h-6 object-contain', className)}
        // style={{ width: '22px' }}
      >
        <FileIcon extension=".html" {...defaultStyles.html} />
      </div>
    );
  else {
    var fileExt = filename.split('.').pop();
    return (
      <div
        className={cn('w-5 h-6 object-contain', className)}
        // style={{ width: '22px' }}
      >
        <FileIcon extension={fileExt} {...defaultStyles[fileExt]} width={1} />
      </div>
    );
  }
};
