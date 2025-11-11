import { IBlueBorderButton, ICancelButton } from '@/app/components/Form/Button';
import { Input } from '@/app/components/Form/Input';
import { cn } from '@/styles/media';
import { Plus, Trash2 } from 'lucide-react';
import { FC } from 'react';

interface Parameter {
  key: string;
  value: string;
}

export const APiParameter: FC<{
  inputClass?: string;
  initialValues: Parameter[]; // Initial values to display (optional)
  setParameterValue: (params: Parameter[]) => void; // Callback for updated parameters
  actionButtonLabel?: string; // Label for add button
}> = ({
  initialValues,
  setParameterValue,
  inputClass,
  actionButtonLabel = 'Add New Pair',
}) => {
  const updateParameter = (
    index: number,
    field: 'key' | 'value',
    value: string,
  ) => {
    const updatedParameters = [...initialValues];
    updatedParameters[index][field] = value;
    setParameterValue(updatedParameters);
  };

  const removeParameter = (index: number) => {
    const updatedParameters = initialValues.filter((_, i) => i !== index);
    setParameterValue(updatedParameters);
  };

  const addParameter = () => {
    setParameterValue([...initialValues, { key: '', value: '' }]);
  };

  return (
    <>
      <div className="text-sm grid w-full">
        {initialValues.map((parameter, index) => (
          <div
            key={`param-${index}`}
            className="grid grid-cols-2 border-b border-gray-400 dark:border-gray-600"
          >
            <div className="flex col-span-1 items-center border-r">
              <Input
                value={parameter.key}
                onChange={e => updateParameter(index, 'key', e.target.value)}
                placeholder="Key"
                className={cn(
                  'bg-light-background w-full border-none',
                  inputClass,
                )}
              />
            </div>
            <div className="col-span-1 flex">
              <Input
                value={parameter.value}
                onChange={e => updateParameter(index, 'value', e.target.value)}
                placeholder="Value"
                className={cn(
                  'bg-light-background w-full border-none',
                  inputClass,
                )}
              />
              <ICancelButton
                className={cn(
                  '!border-transparent hover:!border-red-600 outline-hidden cursor-pointer hover:!text-red-600',
                  inputClass,
                )}
                onClick={() => removeParameter(index)}
                type="button"
              >
                <Trash2 className="w-4 h-4" strokeWidth={1.5} />
              </ICancelButton>
            </div>
          </div>
        ))}
      </div>
      <IBlueBorderButton
        onClick={addParameter}
        className="justify-between space-x-8"
      >
        <span>{actionButtonLabel}</span> <Plus className="h-4 w-4 ml-1.5" />
      </IBlueBorderButton>
    </>
  );
};
