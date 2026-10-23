import { Metadata } from '@rapidaai/react';
import { TextToSpeechProvider } from '@/app/components/domain/providers/text-to-speech';
import { useCallback, useState } from 'react';
import {
  GetDefaultSpeakerConfig,
  GetDefaultTextToSpeechIfInvalid,
} from '@/app/components/domain/providers/text-to-speech/provider';
import {
  CONJUNCTION_BOUNDARIES,
  PRONUNCIATION_DICTIONARIES,
} from '@/providers';
import { ChevronDown } from '@carbon/icons-react';
import { cn } from '@/utils';
import { TextInput } from '@/app/components/ui/primitives';
import { Text } from '@/app/components/ui/primitives/text';
import { Dropdown, MultiSelect, Slider } from '@carbon/react';

const AMBIENT_OPTIONS = [
  { id: 'none', label: 'None' },
  { id: 'office', label: 'Office' },
  { id: 'cafe', label: 'Cafe' },
  { id: 'calm_studio', label: 'Calm Studio' },
];

interface ConfigureAudioOutputProviderProps {
  audioOutputConfig: { provider: string; parameters: Metadata[] };
  setAudioOutputConfig: (config: {
    provider: string;
    parameters: Metadata[];
  }) => void;
}

export const ConfigureAudioOutputProvider: React.FC<
  ConfigureAudioOutputProviderProps
> = ({ audioOutputConfig, setAudioOutputConfig }) => {
  const [showAdvanced, setShowAdvanced] = useState(false);

  const onChangeAudioOutputProvider = (providerName: string) => {
    const parametersToKeep = audioOutputConfig.parameters.filter(p =>
      [
        'speaker.ambient',
        'speaker.ambient_volume',
        'speaker.conjunction.boundaries',
        'speaker.conjunction.break',
        'speaker.pronunciation.dictionaries',
      ].includes(p.getKey()),
    );
    setAudioOutputConfig({
      provider: providerName,
      parameters: GetDefaultTextToSpeechIfInvalid(
        providerName,
        GetDefaultSpeakerConfig(parametersToKeep),
      ),
    });
  };

  const onChangeAudioOutputParameter = (parameters: Metadata[]) => {
    if (audioOutputConfig)
      setAudioOutputConfig({ ...audioOutputConfig, parameters });
  };

  const getParamValue = useCallback(
    (key: string, defaultValue: any) => {
      const param = audioOutputConfig.parameters?.find(p => p.getKey() === key);
      return param ? param.getValue() : defaultValue;
    },
    [audioOutputConfig.parameters],
  );

  const updateParameter = (key: string, value: string) => {
    const updatedParams = (audioOutputConfig.parameters || []).map(param => {
      if (param.getKey() === key) {
        const updatedParam = new Metadata();
        updatedParam.setKey(key);
        updatedParam.setValue(value);
        return updatedParam;
      }
      return param;
    });
    if (!updatedParams.some(param => param.getKey() === key)) {
      const newParam = new Metadata();
      newParam.setKey(key);
      newParam.setValue(value);
      updatedParams.push(newParam);
    }
    onChangeAudioOutputParameter(updatedParams);
  };

  const pronunciationItems = PRONUNCIATION_DICTIONARIES.map(d => ({
    id: d,
    label: d,
  }));
  const conjunctionItems = CONJUNCTION_BOUNDARIES.map(b => ({
    id: b,
    label: b,
  }));

  const selectedPronunciations = getParamValue(
    'speaker.pronunciation.dictionaries',
    '',
  )
    .split('<|||>')
    .filter(Boolean);
  const selectedConjunctions = getParamValue(
    'speaker.conjunction.boundaries',
    '',
  )
    .split('<|||>')
    .filter(Boolean);
  const ambientVolumeRaw = getParamValue('speaker.ambient_volume', '18');
  const ambientVolumeParsed = Number.parseInt(ambientVolumeRaw, 10);
  const ambientVolumeValue = Number.isNaN(ambientVolumeParsed)
    ? 18
    : Math.min(Math.max(ambientVolumeParsed, 0), 100);

  return (
    <div className="flex flex-col gap-6 max-w-4xl p-6">
      <TextToSpeechProvider
        onChangeProvider={onChangeAudioOutputProvider}
        onChangeParameter={onChangeAudioOutputParameter}
        provider={audioOutputConfig.provider}
        parameters={audioOutputConfig.parameters}
      />

      {audioOutputConfig.provider && (
        <>
          <button
            type="button"
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="flex items-center gap-1.5 text-[var(--cds-text-secondary)] transition-colors hover:text-[var(--cds-text-primary)]"
          >
            <ChevronDown
              size={16}
              className={cn(
                'transition-transform duration-200',
                showAdvanced && 'rotate-180',
              )}
            />
            <Text as="span" variant="body-compact-01">
              {showAdvanced ? 'Hide' : 'Show'} advanced settings
            </Text>
          </button>

          {showAdvanced && (
            <div className="flex flex-col gap-6 pt-6 border-t border-gray-200 dark:border-gray-800">
              <Text
                as="p"
                variant="label-01"
                className="uppercase text-[var(--cds-text-secondary)]"
              >
                Ambient
              </Text>
              <Dropdown
                id="ambient-sound"
                titleText="Ambient"
                label="Select ambient sound"
                items={AMBIENT_OPTIONS}
                selectedItem={
                  AMBIENT_OPTIONS.find(
                    option =>
                      option.id === getParamValue('speaker.ambient', 'none'),
                  ) || AMBIENT_OPTIONS[0]
                }
                itemToString={(item: any) => item?.label || ''}
                onChange={({ selectedItem }: any) =>
                  updateParameter('speaker.ambient', selectedItem?.id || 'none')
                }
              />
              <Slider
                id="ambient-volume"
                labelText="Ambient Volume"
                min={0}
                max={100}
                step={1}
                value={ambientVolumeValue}
                onChange={({ value }: { value: number }) =>
                  updateParameter('speaker.ambient_volume', value.toString())
                }
              />

              <Text
                as="p"
                variant="label-01"
                className="uppercase text-[var(--cds-text-secondary)]"
              >
                Experience
              </Text>
              <MultiSelect
                id="pronunciation-dictionaries"
                titleText="Pronunciation Dictionaries"
                label="Select all that applies"
                items={pronunciationItems}
                selectedItems={pronunciationItems.filter(i =>
                  selectedPronunciations.includes(i.id),
                )}
                itemToString={(item: any) => item?.label || ''}
                onChange={({ selectedItems }: any) =>
                  updateParameter(
                    'speaker.pronunciation.dictionaries',
                    selectedItems.map((i: any) => i.id).join('<|||>'),
                  )
                }
                helperText="Pronunciation dictionaries help define custom pronunciations for words, abbreviations, and acronyms."
              />
              <MultiSelect
                id="conjunction-boundaries"
                titleText="Conjunction Boundaries"
                label="Select all that applies"
                items={conjunctionItems}
                selectedItems={conjunctionItems.filter(i =>
                  selectedConjunctions.includes(i.id),
                )}
                itemToString={(item: any) => item?.label || ''}
                onChange={({ selectedItems }: any) =>
                  updateParameter(
                    'speaker.conjunction.boundaries',
                    selectedItems.map((i: any) => i.id).join('<|||>'),
                  )
                }
                helperText="Conjunctions treated as valid boundaries for adding a pause before delivering to the voice provider."
              />

              <TextInput
                id="conjunction-break"
                labelText="Pause Duration (Milliseconds)"
                type="number"
                value={getParamValue('speaker.conjunction.break', '240')}
                onChange={e =>
                  updateParameter('speaker.conjunction.break', e.target.value)
                }
              />
            </div>
          )}
        </>
      )}
    </div>
  );
};
