import { Metadata } from '@rapidaai/react';
import { Dropdown } from '@/app/components/Dropdown';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { InputGroup } from '@/app/components/input-group';
import { InputHelper } from '@/app/components/input-helper';
import { ProviderConfig } from '@/app/components/providers';

import { cn } from '@/styles/media';
import { TextToSpeechProvider } from '@/app/components/providers/text-to-speech';
import { ConditionalInputGroup } from '@/app/components/conditional-input-group';

/**
 *
 * @param param0
 * @returns
 */
export const ConfigureAudioOutputProvider: React.FC<{
  onChangeProvider: (providerId: string, providerName: string) => void;
  onChangeConfig: (config: ProviderConfig) => void;
  config: ProviderConfig | null;
  voiceOutputEnable: boolean;
  onChangeVoiceOutputEnable: (b: boolean) => void;
}> = ({
  onChangeProvider,
  onChangeConfig,
  config,
  voiceOutputEnable,
  onChangeVoiceOutputEnable,
}) => {
  const updateConfig = (newConfig: Partial<ProviderConfig>) => {
    onChangeConfig({ ...config, ...newConfig } as ProviderConfig);
  };

  return (
    <ConditionalInputGroup
      title="Voice Output"
      enable={voiceOutputEnable}
      onChangeEnable={onChangeVoiceOutputEnable}
    >
      <div className={cn('px-6 pb-6 pt-2 flex gap-8 pl-8')}>
        <TextToSpeechProvider
          config={config}
          onChangeConfig={onChangeConfig}
          onChangeProvider={onChangeProvider}
        />
      </div>
      {config?.provider && (
        <ConfigureSpeakerExperience
          speakConfig={config.parameters}
          onChangeConfig={updateConfig}
        />
      )}
    </ConditionalInputGroup>
  );
};

export const ConfigureSpeakerExperience: React.FC<{
  speakConfig: Metadata[];
  onChangeConfig: (m: Partial<ProviderConfig>) => void;
}> = ({ speakConfig, onChangeConfig }) => {
  //
  const getParamValue = (key: string, defaultValue: any) =>
    speakConfig?.find(p => p.getKey() === key)?.getValue() ?? defaultValue;

  //
  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...speakConfig];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }
    onChangeConfig({ parameters: updatedParams });
  };

  return (
    <InputGroup initiallyExpanded={false} title="Speaker Expereience">
      <div className={cn('px-6 pb-6 pt-2 flex gap-8 pl-8')}>
        <div className="space-y-6 w-full max-w-6xl">
          <FieldSet className="relative">
            <FormLabel>Sentence Boundaries</FormLabel>
            <Dropdown
              multiple
              className="bg-light-background dark:bg-gray-950 max-w-6xl"
              currentValue={getParamValue(
                'speaker.sentence.boundaries',
                [
                  '.', // Period
                  '!', // Exclamation mark
                  '?', // Question mark
                  '|', // Pipe
                  ';', // Semicolon
                  ':', // Colon
                  '…', // Ellipsis
                  '。', // Chinese/Japanese full stop
                  '．', // Katakana middle dot
                  '।', // Devanagari Danda (Hindi full stop)
                  '۔', // Arabic full stop
                  '--', // Double dash
                ].join('<|||>'),
              ).split('<|||>')}
              setValue={v => {
                updateParameter('speaker.sentence.boundaries', v.join('<|||>'));
              }}
              allValue={[
                '.', // Period
                '!', // Exclamation mark
                '?', // Question mark
                '|', // Pipe
                ';', // Semicolon
                ':', // Colon
                '…', // Ellipsis
                '。', // Chinese/Japanese full stop
                '．', // Katakana middle dot
                '।', // Devanagari Danda (Hindi full stop)
                '۔', // Arabic full stop
                '--', // Double dash
              ]}
              placeholder="Select all that applies"
              option={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    <span className="truncate capitalize">{c}</span>
                  </span>
                );
              }}
              label={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    {c.map(x => {
                      return (
                        <span key={x} className="truncate">
                          {x}
                        </span>
                      );
                    })}
                  </span>
                );
              }}
            />
            <InputHelper>
              These are the sentence that are considered valid boundaries or
              delimiters. This helps decides the chunks that are sent to the
              voice provider for the voice generation as the LLM tokens are
              streaming in
            </InputHelper>
          </FieldSet>
          <FieldSet className="relative">
            <FormLabel>Conjunction Boundaries</FormLabel>
            <Dropdown
              multiple
              className="bg-light-background dark:bg-gray-950 max-w-6xl"
              currentValue={getParamValue(
                'speaker.conjunction.boundaries',
                [
                  'for',
                  'and',
                  'nor',
                  'but',
                  'or',
                  'yet',
                  'so',
                  'after',
                  'although',
                  'as',
                  'because',
                  'before',
                  'even',
                  'if',
                  'once',
                  'since',
                  'so that',
                  'than',
                  'that',
                  'though',
                  'unless',
                  'until',
                  'when',
                  'whenever',
                  'where',
                  'wherever',
                  'whereas',
                  'whether',
                  'while',
                ].join('<|||>'),
              ).split('<|||>')}
              setValue={v => {
                updateParameter(
                  'speaker.conjunction.boundaries',
                  v.join('<|||>'),
                );
              }}
              allValue={[
                'for',
                'and',
                'nor',
                'but',
                'or',
                'yet',
                'so',
                'after',
                'although',
                'as',
                'because',
                'before',
                'even',
                'if',
                'once',
                'since',
                'so that',
                'than',
                'that',
                'though',
                'unless',
                'until',
                'when',
                'whenever',
                'where',
                'wherever',
                'whereas',
                'whether',
                'while',
              ]}
              placeholder="Select all that applies"
              option={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    <span className="truncate capitalize">{c}</span>
                  </span>
                );
              }}
              label={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    {c.map(x => {
                      return (
                        <span key={x} className="truncate">
                          {x}
                        </span>
                      );
                    })}
                  </span>
                );
              }}
            />
            <InputHelper>
              These are the punctuations that are considered valid boundaries or
              delimiters. This helps decides to add pause before delivering to
              voice provider
            </InputHelper>
          </FieldSet>
          <FieldSet className="relative">
            <FormLabel>Pronunciation Dictionaries</FormLabel>
            <Dropdown
              multiple
              className="bg-light-background dark:bg-gray-950 max-w-6xl"
              currentValue={getParamValue(
                'speaker.pronunciation.dictionaries',
                [
                  'currency',
                  'date',
                  'time',
                  'numeral',
                  'address',
                  'url',
                  'tech-abbreviation',
                  'role-abbreviation',
                  'general-abbreviation',
                  'symbol',
                ].join('<|||>'),
              ).split('<|||>')}
              setValue={v => {
                updateParameter(
                  'speaker.pronunciation.dictionaries',
                  v.join('<|||>'),
                );
              }}
              allValue={[
                'currency',
                'date',
                'time',
                'numeral',
                'address',
                'url',
                'tech-abbreviation',
                'role-abbreviation',
                'general-abbreviation',
                'symbol',
              ]}
              placeholder="Select all that applies"
              option={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    <span className="truncate capitalize">{c}</span>
                  </span>
                );
              }}
              label={c => {
                return (
                  <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                    {c.map(x => {
                      return (
                        <span key={x} className="truncate">
                          {x}
                        </span>
                      );
                    })}
                  </span>
                );
              }}
            />
            <InputHelper>
              Pronunciation dictionaries help define custom pronunciations for
              words, abbreviations, and acronyms. They ensure correct
              pronunciation of domain-specific terms, names, or technical jargon
              that may not be pronounced correctly by default text-to-speech
              systems.
            </InputHelper>
          </FieldSet>
        </div>
      </div>
    </InputGroup>
  );
};
