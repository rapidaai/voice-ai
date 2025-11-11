import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const GOOGLE_LANGUAGE = [
  { code: 'en-SG', name: 'English (Singapore)' },
  { code: 'en-US', name: 'English (United States)' },
  { code: 'hi-IN', name: 'Hindi (India)' },
  { code: 'gu-IN', name: 'Gujarati (India)' },
  { code: 'kn-IN', name: 'Kannada (India)' },
  { code: 'ta-IN', name: 'Tamil (India)' },
  { code: 'ms-MY', name: 'Malay (Malaysia)' },
  { code: 'ml-IN', name: 'Malayalam (India)' },
  { code: 'en-IN', name: 'English (India)' },
  { code: 'cmn-Hans-CN', name: 'Chinese (Simplified, China)' },
  { code: 'th-TH', name: 'Thai (Thailand)' },
  { code: 'id-ID', name: 'Indonesian (Indonesia)' },
];

export const GOOGLE_MODELS = [
  {
    name: 'default',
    id: 'default',
  },
  {
    name: 'command_and_search',
    id: 'command_and_search',
  },
  {
    name: 'telephony',
    id: 'telephony',
  },
];
// export const GOOGLE_ENCODINGS = [
//   { value: 'unspecified', name: 'Not specified' },
//   { value: 'linear16', name: 'Linear16' },
//   { value: 'flac', name: 'FLAC' },
//   { value: 'mulaw', name: 'Mulaw' },
//   { value: 'amr', name: 'AMR' },
//   { value: 'amr_wb', name: 'AMR-WB' },
//   { value: 'ogg_opus', name: 'OGG_OPUS' },
//   { value: 'speex', name: 'SPEEX' },
//   { value: 'mp3', name: 'MP3 (Beta)' },
//   { value: 'webm_opus', name: 'WEBM_OPUS' },
// ];

// export const GOOGLE_SAMPLE_RATES = [
//   { value: '8000', name: '8000 Hz' },
//   { value: '16000', name: '16000 Hz' },
//   { value: '24000', name: '24000 Hz' },
//   { value: '32000', name: '32000 Hz' },
//   { value: '48000', name: '48000 Hz' },
// ];

export const GetGoogleDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];
  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'listen.language',
    'listen.model',
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
    GOOGLE_LANGUAGE.some(l => l.code === value),
  );

  // Set model
  addMetadata('listen.model', 'default', value =>
    GOOGLE_MODELS.some(m => m.id === value),
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
    !GOOGLE_LANGUAGE.some(lang => lang.code === languageOption.getValue())
  ) {
    return false;
  }

  // Validate model
  const modelOption = options.find(opt => opt.getKey() === 'listen.model');
  if (
    !modelOption ||
    !GOOGLE_MODELS.some(m => m.id === modelOption.getValue())
  ) {
    return false;
  }
  return true;
};
