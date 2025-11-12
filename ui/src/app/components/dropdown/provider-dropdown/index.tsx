import { Dropdown } from '@/app/components/dropdown';
import { COMPLETE_PROVIDER, RapidaProvider } from '@/app/components/providers';

export function ProviderDropdown(props: {
  currentProvider?: RapidaProvider;
  setCurrentProvider: (v: RapidaProvider) => void;
}) {
  //   const { modelProviders } = useAllProviders();
  //   const [providers, setProviders] = useState<Provider[]>(modelProviders);

  //   useEffect(() => {
  //     setProviders(modelProviders);
  //   }, [modelProviders]);

  const dropdownItem = (p: RapidaProvider) => {
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
  };
  return (
    <Dropdown
      currentValue={props.currentProvider}
      setValue={props.setCurrentProvider}
      allValue={COMPLETE_PROVIDER}
      className="bg-white dark:bg-gray-950"
      placeholder="Select the provider"
      label={dropdownItem}
      option={dropdownItem}
    />
  );
}
