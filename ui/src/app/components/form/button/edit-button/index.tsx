import { ButtonProps, IconButton } from '@/app/components/form/button';
import { cn } from '@/utils';
import { FilePenLine } from 'lucide-react';
import { FC } from 'react';

export const EditButton: FC<ButtonProps> = props => {
  return (
    <IconButton
      className={cn('hover:bg-blue-600/10! hover:text-blue-600!')}
      onClick={props.onClick}
    >
      <FilePenLine className="w-4 h-4" />
    </IconButton>
  );
};
