import { cn } from '@/utils';
import { Dropdown } from '@/app/components/dropdown';
import {
  INTEGRATION_PROVIDER,
  IntegrationProvider,
} from '@/app/components/providers';

export function ExternalProviderDropdown(props: {
  externalProvider?: IntegrationProvider;
  setExternalProvider: (v: IntegrationProvider) => void;
}) {
  return (
    <Dropdown
      allValue={INTEGRATION_PROVIDER}
      currentValue={props.externalProvider}
      setValue={props.setExternalProvider}
      className="bg-white dark:bg-gray-950"
      placeholder="Select a provider"
      label={(p: IntegrationProvider) => {
        return (
          <span className="inline-flex items-center gap-1.5 sm:gap-2 max-w-full text-sm font-medium">
            <img
              alt={p.name}
              loading="lazy"
              className="w-5 h-5 align-middle block shrink-0"
              src={p.image}
            />
            <span className="truncate capitalize">{p.name}</span>
          </span>
        );
      }}
      option={(prj, selected) => {
        return (
          <span className="inline-flex items-center gap-1.5 sm:gap-2 max-w-full text-sm font-medium">
            <img
              alt={prj.name}
              loading="lazy"
              className="w-5 h-5 align-middle block shrink-0"
              src={prj.image}
            />
            <span
              className={cn(
                'truncate capitalize',
                selected ? 'opacity-100 font-medium' : 'opacity-80',
              )}
            >
              {prj.name}
            </span>
          </span>
        );
      }}
    />
  );
}
