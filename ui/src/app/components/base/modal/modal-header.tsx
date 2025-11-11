import { IButton } from '@/app/components/Form/Button';
import { cn } from '@/styles/media';
import { X } from 'lucide-react';
import { FC, HTMLAttributes } from 'react';

export const ModalHeader: FC<
  HTMLAttributes<HTMLDivElement> & {
    onClose: () => void;
  }
> = props => {
  return (
    <div
      className={cn('flex flex-col relative pt-6 pb-3 px-6', props.className)}
      {...props}
    >
      {props.children}
      <IButton onClick={props.onClose} className="absolute top-0 right-0">
        <X className="w-5 h-5" />
      </IButton>
    </div>
  );
};
