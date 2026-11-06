import { IconOnlyButton } from '@/app/components/ui/primitives/button';
import { Copy, Checkmark } from '@carbon/icons-react';
import type { ReactNode } from 'react';
import { useState } from 'react';
import { cn } from '@/utils';

interface CopyButtonProps {
  children?: ReactNode;
  className?: string;
  copyDescription?: string;
  copiedDescription?: string;
}

export function CopyButton({
  children,
  className,
  copyDescription = 'Copy',
  copiedDescription = 'Copied',
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const copyItem = (item: ReactNode) => {
    setCopied(true);
    navigator.clipboard.writeText(String(item ?? ''));
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <IconOnlyButton
      kind="ghost"
      size="sm"
      renderIcon={copied ? Checkmark : Copy}
      iconDescription={copied ? copiedDescription : copyDescription}
      onClick={() => copyItem(children)}
      className={cn(copied && 'text-support-success', className)}
    />
  );
}
