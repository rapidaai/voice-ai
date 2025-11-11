import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { GOOGLE_EMBEDDING_MODEL } from '@/app/components/providers/embedding/google/constants';

export const ConfigureGoogleEmbeddingModel: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  return (
    <Dropdown
      className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none!"
      currentValue={GOOGLE_EMBEDDING_MODEL.find(
        x =>
          x.id === getParamValue('model.id') &&
          getParamValue('model.name') === x.name,
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
      allValue={GOOGLE_EMBEDDING_MODEL}
      placeholder="Select embedding model"
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
  );
};
