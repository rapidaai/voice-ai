import React, { FC, memo } from 'react';
import { cn } from '@/styles/media';
import { getIcon } from 'material-file-icons';

interface FileIconProps extends React.SVGProps<SVGSVGElement> {
  filename?: string;
}

export const FileIcon: FC<FileIconProps> = props => {
  const { filename, ...svgProps } = props;
  if (filename)
    return (
      <GenerateFileIcon filename={filename} className={svgProps.className} />
    );
  return (
    <svg
      {...svgProps}
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn('w-5 h-5 opacity-75', svgProps.className)}
    >
      <path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z" />
      <path d="M14 2v4a2 2 0 0 0 2 2h4" />
      <path d="M10 9H8" />
      <path d="M16 13H8" />
      <path d="M16 17H8" />
    </svg>
  );
};

export const GenerateFileIcon: FC<{ filename: string; className?: string }> =
  memo(({ filename, className }) => {
    return (
      <div
        className={className}
        dangerouslySetInnerHTML={{ __html: getIcon(filename).svg }}
      />
    );
  });
