import { Metadata } from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { InputHelper } from '@/app/components/input-helper';

export const ConfigureTwilioWhatsapp: React.FC<{
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
        <FormLabel>Account SID</FormLabel>
        <Input
          className="bg-light-background"
          value={getParamValue('account_sid')}
          placeholder="Enter your Twilio Account SID"
          onChange={v => {
            updateParameter('account_sid', v.target.value);
          }}
        />
      </FieldSet>
      <FieldSet className="col-span-2">
        <FormLabel>Auth Token</FormLabel>
        <Input
          className="bg-light-background"
          value={getParamValue('auth_token')}
          placeholder="Enter your Twilio Auth Token"
          onChange={v => {
            updateParameter('auth_token', v.target.value);
          }}
        />
      </FieldSet>
      <FieldSet className="col-span-2">
        <FormLabel>Phone</FormLabel>
        <Input
          className="bg-light-background"
          value={getParamValue('phone')}
          placeholder="Enter your Twilio phone number"
          onChange={v => {
            updateParameter('phone', v.target.value);
          }}
        />
        <InputHelper>
          Phone to recieve inbound or make outbound call.
        </InputHelper>
      </FieldSet>
    </>
  );
};
