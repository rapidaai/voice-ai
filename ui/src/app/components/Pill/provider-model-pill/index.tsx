import { cn } from '@/styles/media';
import { FC, HTMLAttributes, useEffect, useState } from 'react';
import { COMPLETE_PROVIDER, RapidaProvider } from '@/app/components/providers';

interface ProviderPillProps extends HTMLAttributes<HTMLSpanElement> {
  providerId?: string;
  providerName?: string;
}
export const ProviderPill: FC<ProviderPillProps> = props => {
  //
  const [currentProvider, setcurrentProvider] = useState<RapidaProvider | null>(
    null,
  );

  useEffect(() => {
    if (props.providerId) {
      let cModel = COMPLETE_PROVIDER.find(x => x.id === props.providerId);
      if (cModel) setcurrentProvider(cModel);
    }

    if (props.providerName) {
      let cModel = COMPLETE_PROVIDER.find(
        x => x.name.toLowerCase() === props.providerName?.toLowerCase(),
      );
      if (cModel) setcurrentProvider(cModel);
    }
  }, [props.providerId, COMPLETE_PROVIDER, props.providerName]);

  return (
    <span
      onClick={props.onClick}
      className={cn(
        'px-2 py-1 truncate',
        'items-center bg-blue-400/10 dark:bg-blue-800/10 rounded-[2px]',
        'ring-blue-500/30 dark:ring-blue-400/20 ring-[0.7px]',
        'flex items-center justify-center w-fit',
        props.className,
      )}
    >
      <img
        alt={currentProvider?.name}
        src={currentProvider?.image}
        className="w-4 h-4 mr-1.5 inline-block"
      />
      <span className="font-medium opacity-80 ">{currentProvider?.name}</span>
    </span>
  );
};
