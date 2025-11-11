import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

// export const CARTESIA_SAMPLE_RATES = [
//   {
//     name: '8000 Hz',
//     value: '8000',
//   },
//   {
//     name: '16000 Hz',
//     value: '16000',
//   },
//   {
//     name: '24000 Hz',
//     value: '24000',
//   },
//   {
//     name: '32000 Hz',
//     value: '32000',
//   },
//   {
//     name: '48000 Hz',
//     value: '48000',
//   },
// ];
// export const CARTESIA_ENCODINGS = [
//   {
//     name: 'PCM 32-bit float (Little Endian)',
//     id: 'PCM_F32LE',
//     value: 'pcm_f32le',
//   },
//   {
//     name: 'PCM 16-bit signed integer (Little Endian)',
//     id: 'PCM_S16LE',
//     value: 'pcm_s16le',
//   },
//   {
//     name: 'PCM mu-law',
//     id: 'PCM_MULAW',
//     value: 'pcm_mulaw',
//   },
//   {
//     name: 'PCM A-law',
//     id: 'PCM_ALAW',
//     value: 'pcm_alaw',
//   },
// ];
export const CARTESIA_MODELS = [
  {
    name: 'ink-whisper',
    id: 'ink-whisper',
  },
];
export const CARTESIA_LANGUAGE = [
  { code: 'en', name: 'English' },
  { code: 'fr', name: 'French' },
  { code: 'de', name: 'German' },
  { code: 'es', name: 'Spanish' },
  { code: 'pt', name: 'Portuguese' },
  { code: 'zh', name: 'Chinese' },
  { code: 'ja', name: 'Japanese' },
  { code: 'hi', name: 'Hindi' },
  { code: 'it', name: 'Italian' },
  { code: 'ko', name: 'Korean' },
  { code: 'nl', name: 'Dutch' },
  { code: 'pl', name: 'Polish' },
  { code: 'ru', name: 'Russian' },
  { code: 'sv', name: 'Swedish' },
  { code: 'tr', name: 'Turkish' },
];

export const GetCartesiaDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];

  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'listen.language',
    'listen.model',
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
    CARTESIA_LANGUAGE.some(l => l.code === value),
  );

  // Set model
  addMetadata('listen.model', 'ink-whisper', value =>
    CARTESIA_MODELS.some(m => m.id === value),
  );

  // Only return metadata for the keys we want to keep
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('microphone.')),
  ];
};

export const ValidateCartesiaOptions = (options: Metadata[]): boolean => {
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
    !CARTESIA_LANGUAGE.some(lang => lang.code === languageOption.getValue())
  ) {
    return false;
  }

  // Validate model
  const modelOption = options.find(opt => opt.getKey() === 'listen.model');
  if (
    !modelOption ||
    !CARTESIA_MODELS.some(model => model.id === modelOption.getValue())
  ) {
    return false;
  }

  return true;
};

// ... rest of the code ...
