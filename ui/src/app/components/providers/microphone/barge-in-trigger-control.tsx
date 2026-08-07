import React from 'react';
import { Metadata } from '@rapidaai/react';
import { Select as CarbonSelect, SelectItem } from '@carbon/react';
import { FormLabel } from '@/app/components/form-label';
import { HelpToggletip } from '@/app/components/providers/help-label';

export const MICROPHONE_BARGE_IN_TRIGGER_KEY = 'microphone.barge_in_trigger';
export const LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY =
  'microphone.vad.barge_in_trigger';

const BARGE_IN_TRIGGER_HELP_TEXT =
  'Controls when dispatch allows user barge-in. VAD interrupts on speech start from the VAD detector. Word waits for a recognized word/STT interruption signal before interrupting.';

const BARGE_IN_TRIGGER_CHOICES = [
  { label: 'VAD', value: 'vad' },
  { label: 'Word', value: 'word' },
];

export const BargeInTriggerControl: React.FC<{
  parameters: Metadata[];
  onChangeParameter: (parameters: Metadata[]) => void;
}> = ({ parameters, onChangeParameter }) => {
  const value =
    parameters.find(p => p.getKey() === MICROPHONE_BARGE_IN_TRIGGER_KEY)
      ?.getValue() ??
    parameters
      .find(p => p.getKey() === LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY)
      ?.getValue() ??
    'vad';

  const updateValue = (nextValue: string) => {
    const nextParam = new Metadata();
    nextParam.setKey(MICROPHONE_BARGE_IN_TRIGGER_KEY);
    nextParam.setValue(nextValue);

    onChangeParameter([
      ...parameters.filter(
        p =>
          p.getKey() !== MICROPHONE_BARGE_IN_TRIGGER_KEY &&
          p.getKey() !== LEGACY_MICROPHONE_VAD_BARGE_IN_TRIGGER_KEY,
      ),
      nextParam,
    ]);
  };

  return (
    <div className="min-w-0">
      <span className="inline-flex items-center gap-1">
        <FormLabel htmlFor="microphone-barge-in-trigger">
          Barge-in Trigger
        </FormLabel>
        <HelpToggletip
          label="Barge-in Trigger"
          helpText={BARGE_IN_TRIGGER_HELP_TEXT}
        />
      </span>
      <CarbonSelect
        id="microphone-barge-in-trigger"
        labelText=""
        value={value}
        onChange={e => updateValue(e.target.value)}
      >
        {BARGE_IN_TRIGGER_CHOICES.map(choice => (
          <SelectItem
            key={choice.value}
            value={choice.value}
            text={choice.label}
          />
        ))}
      </CarbonSelect>
    </div>
  );
};
