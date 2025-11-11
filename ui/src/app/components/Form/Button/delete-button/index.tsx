import { ButtonProps, IconButton } from '@/app/components/Form/Button';
import { cn } from '@/styles/media';
import { Trash2 } from 'lucide-react';
import { FC } from 'react';

export const DeleteButton: FC<ButtonProps> = props => {
  return (
    <IconButton
      className={cn('hover:bg-red-600/10! hover:text-red-600!')}
      onClick={props.onClick}
    >
      <Trash2 className="w-4 h-4" />
    </IconButton>
  );
};
