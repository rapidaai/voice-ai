import { Metadata } from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';

export const ValidateAwsStorageOptions = (options: Metadata[]): boolean => {
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
  const s3_bucket_name = options.find(opt => opt.getKey() === 's3_bucket_name');
  if (
    !s3_bucket_name ||
    !s3_bucket_name.getValue() ||
    s3_bucket_name.getValue().length === 0
  ) {
    return false;
  }
  return true;
};

export const ConfigureAwsStorage: React.FC<{
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
        <FormLabel>S3 Bucket Name</FormLabel>
        <Input
          type="text"
          value={getParamValue('s3_bucket_name')}
          onChange={v => updateParameter('s3_bucket_name', v.target.value)}
          className="bg-light-background"
          placeholder="Enter your S3 bucket name"
        />
      </FieldSet>
    </>
  );
};
