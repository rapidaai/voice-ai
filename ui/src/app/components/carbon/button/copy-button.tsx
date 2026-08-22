import { IconOnlyButton } from '@/app/components/carbon/button';
import { Copy, Checkmark } from '@carbon/icons-react';
import type { FC, ReactNode } from 'react';
import { useState } from 'react';
import { cn } from '@/utils';

interface CopyButtonProps {
  children?: ReactNode;
  className?: string;
  copyDescription?: string;
  copiedDescription?: string;
}

export const CopyButton: FC<CopyButtonProps> = ({
  children,
  className,
  copyDescription = 'Copy',
  copiedDescription = 'Copied',
}) => {
  const [copied, setCopied] = useState(false);

  const copyItem = (item: any) => {
    setCopied(true);
    navigator.clipboard.writeText(item);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <IconOnlyButton
      kind="ghost"
      size="sm"
      renderIcon={copied ? Checkmark : Copy}
      iconDescription={copied ? copiedDescription : copyDescription}
      onClick={() => copyItem(children)}
      className={cn(copied && 'text-green-600', className)}
    />
  );
};
