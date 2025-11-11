import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const ASSEMBLYAI_LANGUAGE = [
  { code: 'af', name: 'Afrikaans' },
  { code: 'am', name: 'Amharic' },
  { code: 'ar', name: 'Arabic' },
  { code: 'as', name: 'Assamese' },
  { code: 'az', name: 'Azerbaijani' },
  { code: 'bg', name: 'Bulgarian' },
  { code: 'bn', name: 'Bangla' },
  { code: 'bs', name: 'Bosnian' },
  { code: 'ca', name: 'Catalan' },
  { code: 'cs', name: 'Czech' },
  { code: 'cy', name: 'Welsh' },
  { code: 'da', name: 'Danish' },
  { code: 'de', name: 'German' },
  { code: 'el', name: 'Greek' },
  { code: 'en', name: 'English' },
  { code: 'es', name: 'Spanish' },
  { code: 'et', name: 'Estonian' },
  { code: 'eu', name: 'Basque' },
  { code: 'fa', name: 'Persian' },
  { code: 'fi', name: 'Finnish' },
  { code: 'fil', name: 'Filipino' },
  { code: 'fr', name: 'French' },
  { code: 'ga', name: 'Irish' },
  { code: 'gl', name: 'Galician' },
  { code: 'gu', name: 'Gujarati' },
  { code: 'he', name: 'Hebrew' },
  { code: 'hi', name: 'Hindi' },
  { code: 'hr', name: 'Croatian' },
  { code: 'hu', name: 'Hungarian' },
  { code: 'hy', name: 'Armenian' },
  { code: 'id', name: 'Indonesian' },
  { code: 'is', name: 'Icelandic' },
  { code: 'it', name: 'Italian' },
  { code: 'ja', name: 'Japanese' },
  { code: 'jv', name: 'Javanese' },
  { code: 'ka', name: 'Georgian' },
  { code: 'kk', name: 'Kazakh' },
  { code: 'km', name: 'Khmer' },
  { code: 'kn', name: 'Kannada' },
  { code: 'ko', name: 'Korean' },
  { code: 'lo', name: 'Lao' },
  { code: 'lt', name: 'Lithuanian' },
  { code: 'lv', name: 'Latvian' },
  { code: 'mk', name: 'Macedonian' },
  { code: 'ml', name: 'Malayalam' },
  { code: 'mn', name: 'Mongolian' },
  { code: 'mr', name: 'Marathi' },
  { code: 'ms', name: 'Malay' },
  { code: 'mt', name: 'Maltese' },
  { code: 'my', name: 'Burmese' },
  { code: 'nb', name: 'Norwegian Bokmål' },
  { code: 'ne', name: 'Nepali' },
  { code: 'nl', name: 'Dutch' },
  { code: 'or', name: 'Odia' },
  { code: 'pa', name: 'Punjabi' },
  { code: 'pl', name: 'Polish' },
  { code: 'ps', name: 'Pashto' },
  { code: 'pt', name: 'Portuguese' },
  { code: 'ro', name: 'Romanian' },
  { code: 'ru', name: 'Russian' },
  { code: 'si', name: 'Sinhala' },
  { code: 'sk', name: 'Slovak' },
  { code: 'sl', name: 'Slovenian' },
  { code: 'so', name: 'Somali' },
  { code: 'sq', name: 'Albanian' },
  { code: 'sr', name: 'Serbian' },
  { code: 'su', name: 'Sundanese' },
  { code: 'sv', name: 'Swedish' },
  { code: 'sw', name: 'Swahili' },
  { code: 'ta', name: 'Tamil' },
  { code: 'te', name: 'Telugu' },
  { code: 'th', name: 'Thai' },
  { code: 'tr', name: 'Turkish' },
  { code: 'uk', name: 'Ukrainian' },
  { code: 'ur', name: 'Urdu' },
  { code: 'uz', name: 'Uzbek' },
  { code: 'vi', name: 'Vietnamese' },
  { code: 'zh', name: 'Chinese' },
  { code: 'zu', name: 'Zulu' },
];

export const ASSEMBLYAI_MODELS = [
  {
    id: 'slam-1',
    name: 'Slam-1',
    description:
      'Highest accuracy for English pre-recorded audio with fine-tuning support',
    bestFor: 'English content requiring highest accuracy',
  },
  {
    id: 'universal',
    name: 'Universal',
    description:
      'Best for out-of-the-box transcription with multi-lingual support',
    bestFor: 'Production-ready transcription out of the box',
  },
  {
    id: 'nano',
    name: 'Nano',
    description:
      'Most cost-effective transcription with broad language support',
    bestFor: 'Cost-sensitive applications with broad language needs',
  },
  {
    id: 'universal-streaming',
    name: 'Universal-Streaming',
    description: 'Streaming audio transcription for real-time applications',
    bestFor: 'Voice agents and real-time voice applications',
  },
];

// export const ASSEMBLY_ENCODINGS = [
//   { value: 'pcm_s16le', name: 'Linear16' },
//   { value: 'pcm_mulaw', name: 'Mulaw' },
// ];

// export const ASSEMBLY_SAMPLE_RATES = [
//   { value: '8000', name: '8000 Hz' },
//   { value: '16000', name: '16000 Hz' },
//   { value: '24000', name: '24000 Hz' },
//   { value: '32000', name: '32000 Hz' },
//   { value: '48000', name: '48000 Hz' },
// ];

export const GetAssemblyAIDefaultOptions = (
  current: Metadata[],
): Metadata[] => {
  const mtds: Metadata[] = [];

  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'listen.language',
    'listen.model',
    'listen.threshold',
  ];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  // Set language
  addMetadata('listen.language', 'en', value =>
    ASSEMBLYAI_LANGUAGE.some(l => l.code === value),
  );

  // Set model
  addMetadata('listen.model', 'slam-1', value =>
    ASSEMBLYAI_MODELS.some(m => m.id === value),
  );

  // Set threshold
  addMetadata('listen.threshold', '0.5');
  addMetadata('rapida.credential_id');

  // Only return metadata for the keys we want to keep
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('microphone.')),
  ];
};

export const ValidateAssemblyAIOptions = (options: Metadata[]): boolean => {
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
  const languageOption = options.find(
    opt => opt.getKey() === 'listen.language',
  );
  if (
    !languageOption ||
    !ASSEMBLYAI_LANGUAGE.some(lang => lang.code === languageOption.getValue())
  ) {
    return false;
  }

  // Validate model
  const modelOption = options.find(opt => opt.getKey() === 'listen.model');
  if (
    !modelOption ||
    !ASSEMBLYAI_MODELS.some(m => m.id === modelOption.getValue())
  ) {
    return false;
  }

  return true;
};
