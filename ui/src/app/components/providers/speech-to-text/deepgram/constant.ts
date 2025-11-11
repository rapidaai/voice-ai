import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const DEEPGRAM_MODELS = [
  { id: 'nova-2', name: 'Nova 2' },
  {
    id: 'nova-2-general',
    name: 'Nova 2 General',
    description: 'Optimized for everyday audio processing.',
  },
  {
    id: 'nova-2-meeting',
    name: 'Nova 2 Meeting',
    description:
      'Optimized for conference room settings with multiple speakers and a single listen.',
  },
  {
    id: 'nova-2-phonecall',
    name: 'Nova 2 Phone Call',
    description: 'Optimized for low-bandwidth audio phone calls.',
  },
  {
    id: 'nova-2-voicemail',
    name: 'Nova 2 Voicemail',
    description:
      'Optimized for low-bandwidth audio clips with a single speaker. Derived from the phonecall model.',
  },
  {
    id: 'nova-2-finance',
    name: 'Nova 2 Finance',
    description:
      'Optimized for multiple speakers with varying audio quality, such as on earnings calls. Finance-oriented vocabulary.',
  },
  {
    id: 'nova-2-conversationalai',
    name: 'Nova 2 Conversational AI',
    description:
      'Optimized for human-bot interactions like IVR, voice assistants, or automated kiosks.',
  },
  {
    id: 'nova-2-video',
    name: 'Nova 2 Video',
    description: 'Optimized for audio sourced from videos.',
  },
  {
    id: 'nova-2-medical',
    name: 'Nova 2 Medical',
    description: 'Optimized for audio with medical-oriented vocabulary.',
  },
  {
    id: 'nova-2-drivethru',
    name: 'Nova 2 Drive-Thru',
    description: 'Optimized for audio sources from drive-thrus.',
  },
  {
    id: 'nova-2-automotive',
    name: 'Nova 2 Automotive',
    description: 'Optimized for audio with automotive-oriented vocabulary.',
  },
  {
    id: 'nova-2-atc',
    name: 'Nova 2 ATC',
    description: 'Optimized for air traffic control communications.',
  },
  //
  { id: 'nova-3', name: 'Nova 3' },
  { id: 'nova-3-general', name: 'Nova 3 General' },
  { id: 'nova-3-medical', name: 'Nova 3 Medical' },
];

// export const DEEPGRAM_ENCODINGS = [
//   { value: 'linear16', name: 'Linear16' },
//   { value: 'mulaw', name: 'Mulaw' },
//   { value: 'alaw', name: 'Alaw' },
// ];

// export const DEEPGRAM_SAMPLE_RATES = [
//   { value: '8000', name: '8000 Hz' },
//   { value: '16000', name: '16000 Hz' },
//   { value: '24000', name: '24000 Hz' },
//   { value: '32000', name: '32000 Hz' },
//   { value: '48000', name: '48000 Hz' },
// ];

export const DEEPGRAM_LANGUAGES = [
  {
    code: 'multi',
    name: 'Multilingual',
  },
  { code: 'en-US', name: 'English (US)' },
  { code: 'en-AU', name: 'English (Australia)' },
  { code: 'en-GB', name: 'English (UK)' },
  { code: 'en-IN', name: 'English (India)' },
  { code: 'en-NZ', name: 'English (New Zealand)' },
  { code: 'es', name: 'Spanish' },
  { code: 'fr', name: 'French' },
  { code: 'de', name: 'German' },
  { code: 'hi', name: 'Hindi' },
  { code: 'ru', name: 'Russian' },
  { code: 'pt', name: 'Portuguese' },
  { code: 'ja', name: 'Japanese' },
  { code: 'it', name: 'Italian' },
  { code: 'nl', name: 'Dutch' },
  { code: 'ko', name: 'Korean' },
];

export const GetDeepgramDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];

  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'listen.language',
    'listen.model',
    'listen.threshold',
    'listen.keywords',
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
  addMetadata('listen.language', 'multi', value =>
    DEEPGRAM_LANGUAGES.some(l => l.code === value),
  );

  // Set model
  addMetadata('listen.model', 'nova-3', value =>
    DEEPGRAM_MODELS.some(m => m.id === value),
  );

  // Set threshold
  addMetadata('listen.threshold', '0.5');

  // Set keywords
  addMetadata('listen.keywords', '');

  // Only return metadata for the keys we want to keep
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('microphone.')),
  ];
};
export const ValidateDeepgramOptions = (options: Metadata[]): boolean => {
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
    !DEEPGRAM_LANGUAGES.some(lang => lang.code === languageOption.getValue())
  ) {
    return false;
  }

  // Validate model
  const modelOption = options.find(opt => opt.getKey() === 'listen.model');
  if (
    !modelOption ||
    !DEEPGRAM_MODELS.some(model => model.id === modelOption.getValue())
  ) {
    return false;
  }

  return true;
};
