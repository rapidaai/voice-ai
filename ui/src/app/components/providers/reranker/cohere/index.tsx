import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { COHERE_RERANKER_MODEL } from '@/app/components/providers/reranker/cohere/constants';
import { cn } from '@/styles/media';

export const ConfigureCohereRerankerModel: React.FC<{
  inputClass?: string;
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
  disabled?: boolean;
}> = ({ onParameterChange, parameters, disabled, inputClass }) => {
  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...(parameters || [])];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }
    onParameterChange(updatedParams);
  };
  return (
    <div className="flex-1 flex">
      <Dropdown
        disable={disabled}
        className={cn(
          'bg-light-background max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none!',
          inputClass,
        )}
        currentValue={COHERE_RERANKER_MODEL.find(
          x =>
            x.id === getParamValue('model.id') &&
            x.name === getParamValue('model.name'),
        )}
        setValue={v => {
          const updatedParams = [...(parameters || [])];
          const newIdParam = new Metadata();
          const newNameParam = new Metadata();

          newIdParam.setKey('model.id');
          newIdParam.setValue(v.id);
          newNameParam.setKey('model.name');
          newNameParam.setValue(v.name);

          // Remove existing parameters if they exist
          const filteredParams = updatedParams.filter(
            p => p.getKey() !== 'model.id' && p.getKey() !== 'model.name',
          );
          filteredParams.push(newIdParam, newNameParam);
          onParameterChange(filteredParams);
        }}
        allValue={COHERE_RERANKER_MODEL}
        placeholder="Select voice ouput provider"
        option={c => {
          return (
            <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
              <span className="truncate capitalize">{c.name}</span>
            </span>
          );
        }}
        label={c => {
          return (
            <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
              <span className="truncate capitalize">{c.name}</span>
            </span>
          );
        }}
      />
    </div>
  );
};
