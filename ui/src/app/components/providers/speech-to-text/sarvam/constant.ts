import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const SARVAM_LANGUAGE = [
  { code: 'hi-IN', name: 'Hindi' },
  { code: 'bn-IN', name: 'Bengali' },
  { code: 'ta-IN', name: 'Tamil' },
  { code: 'te-IN', name: 'Telugu' },
  { code: 'gu-IN', name: 'Gujarati' },
  { code: 'kn-IN', name: 'Kannada' },
  { code: 'ml-IN', name: 'Malayalam' },
  { code: 'mr-IN', name: 'Marathi' },
  { code: 'pa-IN', name: 'Punjabi' },
  { code: 'od-IN', name: 'Odia' },
  { code: 'en-IN', name: 'English (India)' },
];

export const SARVAM_MODELS = [
  {
    id: 'saarika:v2.5',
    name: 'Saarika',
    type: 'Speech-to-Text (STT)',
    languages_supported: [
      'Hindi',
      'Tamil',
      'Telugu',
      'Kannada',
      'Bengali',
      'Marathi',
      'Gujarati',
      'Punjabi',
      'Malayalam',
      'Odia',
      'English',
    ],
    features: [
      'Real-time and batch transcription',
      'Speaker diarization',
      'Multilingual support',
      'Language auto-detection',
    ],
    use_cases: [
      'Customer support',
      'Voice notes',
      'Business transcription',
      'Accessibility',
    ],
    docs_url:
      'https://docs.sarvam.ai/api-reference-docs/getting-started/models',
  },
  {
    id: 'saaras:v2.5',
    name: 'Saaras',
    type: 'Speech-to-English Translation',
    languages_supported: [
      'Hindi',
      'Tamil',
      'Telugu',
      'Kannada',
      'Bengali',
      'Marathi',
      'Gujarati',
      'Punjabi',
      'Malayalam',
      'Odia',
    ],
    features: [
      'Direct speech-to-English',
      'Entity-aware',
      'Optimized for telephony',
      'Domain-aware translation',
    ],
    use_cases: [
      'Multilingual support desks',
      'Real-time English translation',
      'Call transcription with translation',
    ],
    docs_url:
      'https://docs.sarvam.ai/api-reference-docs/getting-started/models',
  },
];
export const SARVAM_ENCODINGS = [{ value: 'mp3', name: 'MP3 (Beta)' }];

export const SARVAM_SAMPLE_RATES = [
  { value: '8000', name: '8000 Hz' },
  { value: '16000', name: '16000 Hz' },
  { value: '24000', name: '24000 Hz' },
];

export const GetSarvamDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];
  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'listen.language',
    'listen.model',
    'listen.output_format.sample_rate',
    'listen.output_format.encoding',
    'listen.threshold',
  ];

  // Function to create or update metadata
  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  addMetadata('rapida.credential_id');
  // Set language
  addMetadata('listen.language', 'en', value =>
    SARVAM_LANGUAGE.some(l => l.code === value),
  );

  // Set model
  addMetadata('listen.model', 'gemini-2.5-flash', value =>
    SARVAM_MODELS.some(m => m.id === value),
  );

  // Set sample rate
  addMetadata('listen.output_format.sample_rate', '24000', value =>
    SARVAM_SAMPLE_RATES.some(sr => sr.value === value),
  );

  // Set encoding
  addMetadata('listen.output_format.encoding', 'linear16', value =>
    SARVAM_ENCODINGS.some(e => e.value === value),
  );

  addMetadata('listen.threshold', '0.5');

  // Only return metadata for the keys we want to keep
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('microphone.')),
  ];
};

export const ValidateGoogleOptions = (options: Metadata[]): boolean => {
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
    !SARVAM_LANGUAGE.some(lang => lang.code === languageOption.getValue())
  ) {
    return false;
  }

  // Validate model
  const modelOption = options.find(opt => opt.getKey() === 'listen.model');
  if (
    !modelOption ||
    !SARVAM_MODELS.some(m => m.id === modelOption.getValue())
  ) {
    return false;
  }

  return true;
};
