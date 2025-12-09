/**
 * Rapida – Open Source Voice AI Orchestration Platform
 * Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
 * Licensed under a modified GPL-2.0. See the LICENSE file for details.
 */
import { Metadata } from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/form/fieldset';
import { GOOGLE_CLOUD_VOICE } from '@/providers';
import { useState } from 'react';
import { CustomValueDropdown } from '@/app/components/dropdown/custom-value-dropdown';
import { ILinkBorderButton } from '@/app/components/form/button';
import { ExternalLink } from 'lucide-react';
export { GetGoogleDefaultOptions, ValidateGoogleOptions } from './constant';

const renderOption = (c: { icon: React.ReactNode; name: string }) => (
  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
    {c.icon}
    <span className="truncate capitalize">{c.name}</span>
  </span>
);

export const ConfigureGoogleTextToSpeech: React.FC<{
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({ onParameterChange, parameters }) => {
  const allVoices = GOOGLE_CLOUD_VOICE();
  /**
   *
   */
  const [filteredVoices, setFilteredVoices] = useState(allVoices);

  /**
   *
   * @param key
   * @returns
   */
  const getParamValue = (key: string) =>
    parameters?.find(p => p.getKey() === key)?.getValue() ?? '';

  //
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
    <FieldSet className="col-span-1" key="speak.voice.id">
      <FormLabel>Voice</FormLabel>
      <div className="flex">
        <CustomValueDropdown
          searchable
          className="bg-light-background max-w-full dark:bg-gray-950"
          currentValue={filteredVoices.find(
            x => x.name === getParamValue('speak.voice.id'),
          )}
          setValue={(v: { name: string }) =>
            updateParameter('speak.voice.id', v.name)
          }
          allValue={filteredVoices}
          customValue
          onSearching={t => {
            const voices = allVoices;
            const v = t.target.value;
            if (v.length > 0) {
              setFilteredVoices(
                voices.filter(
                  voice =>
                    voice.name.toLowerCase().includes(v.toLowerCase()) ||
                    voice.ssmlGender.toLowerCase().includes(v.toLowerCase()),
                ),
              );
              return;
            }
            setFilteredVoices(voices);
          }}
          onAddCustomValue={vl => {
            updateParameter('speak.voice.id', vl);
          }}
          placeholder="Select voice"
          option={renderOption}
          label={renderOption}
        />
        <ILinkBorderButton
          target="_blank"
          href={`/integration/models/google-cloud?query=${getParamValue('speak.voice.id')}`}
          className="h-10 text-sm p-2 px-3 bg-light-background max-w-full dark:bg-gray-950 border-b"
        >
          <ExternalLink className="w-4 h-4" strokeWidth={1.5} />
        </ILinkBorderButton>
      </div>
    </FieldSet>
  );
};
