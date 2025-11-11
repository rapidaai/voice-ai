import { Metadata } from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';

export const ValidateAzureStorageOptions = (options: Metadata[]): boolean => {
  const credentialID = options.find(
    opt => opt.getKey() === 'rapida.credential_id',
  );
  if (
    !credentialID ||
    !credentialID.getValue() ||
    credentialID.getValue().length === 0
  ) {
    return false;
  }
  // Validate language
  const storage_account_name = options.find(
    opt => opt.getKey() === 'storage_account_name',
  );
  if (
    !storage_account_name ||
    !storage_account_name.getValue() ||
    storage_account_name.getValue().length === 0
  ) {
    return false;
  }

  const container_name = options.find(opt => opt.getKey() === 'container_name');
  if (
    !container_name ||
    !container_name.getValue() ||
    container_name.getValue().length === 0
  ) {
    return false;
  }

  const endpoint_suffix = options.find(
    opt => opt.getKey() === 'endpoint_suffix',
  );
  if (
    !endpoint_suffix ||
    !endpoint_suffix.getValue() ||
    endpoint_suffix.getValue().length === 0
  ) {
    return false;
  }

  return true;
};

export const ConfigureAzureStorage: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  //
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
    <>
      <FieldSet className="col-span-2">
        <FormLabel>Storage Account Name</FormLabel>
        <Input
          type="text"
          value={getParamValue('storage_account_name')}
          onChange={v =>
            updateParameter('storage_account_name', v.target.value)
          }
          className="bg-light-background"
          placeholder="Enter your Azure Storage Account Name"
        />
      </FieldSet>

      <FieldSet className="col-span-2">
        <FormLabel>Container Name</FormLabel>
        <Input
          type="text"
          value={getParamValue('container_name')}
          onChange={v => updateParameter('container_name', v.target.value)}
          className="bg-light-background"
          placeholder="Enter your Azure Container Name"
        />
      </FieldSet>

      <FieldSet className="col-span-2">
        <FormLabel>Endpoint Suffix</FormLabel>
        <Input
          type="text"
          value={getParamValue('endpoint_suffix')}
          onChange={v => updateParameter('endpoint_suffix', v.target.value)}
          className="bg-light-background"
          placeholder="e.g., core.windows.net"
        />
      </FieldSet>
    </>
  );
};
