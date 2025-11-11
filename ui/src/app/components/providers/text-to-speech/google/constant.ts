import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const GOOGLE_LANGUAGE = [
  { code: 'ar-XA', name: 'Arabic (Generic)' },
  { code: 'bn-IN', name: 'Bengali (India)' },
  { code: 'da-DK', name: 'Danish (Denmark)' },
  { code: 'nl-BE', name: 'Dutch (Belgium)' },
  { code: 'nl-NL', name: 'Dutch (Netherlands)' },
  { code: 'en-AU', name: 'English (Australia)' },
  { code: 'en-IN', name: 'English (India)' },
  { code: 'en-GB', name: 'English (United Kingdom)' },
  { code: 'en-US', name: 'English (United States)' },
  { code: 'fi-FI', name: 'Finnish (Finland)' },
  { code: 'fr-CA', name: 'French (Canada)' },
  { code: 'fr-FR', name: 'French (France)' },
  { code: 'de-DE', name: 'German (Germany)' },
  { code: 'gu-IN', name: 'Gujarati (India)' },
  { code: 'hi-IN', name: 'Hindi (India)' },
  { code: 'id-ID', name: 'Indonesian (Indonesia)' },
  { code: 'it-IT', name: 'Italian (Italy)' },
  { code: 'ja-JP', name: 'Japanese (Japan)' },
  { code: 'kn-IN', name: 'Kannada (India)' },
  { code: 'ko-KR', name: 'Korean (South Korea)' },
  { code: 'ml-IN', name: 'Malayalam (India)' },
  { code: 'cmn-CN', name: 'Mandarin Chinese (China)' },
  { code: 'mr-IN', name: 'Marathi (India)' },
  { code: 'nb-NO', name: 'Norwegian Bokmål (Norway)' },
  { code: 'pl-PL', name: 'Polish (Poland)' },
  { code: 'pt-BR', name: 'Portuguese (Brazil)' },
  { code: 'ru-RU', name: 'Russian (Russia)' },
  { code: 'es-ES', name: 'Spanish (Spain)' },
  { code: 'es-US', name: 'Spanish (United States)' },
  { code: 'sw-KE', name: 'Swahili (Kenya)' },
  { code: 'sv-SE', name: 'Swedish (Sweden)' },
  { code: 'ta-IN', name: 'Tamil (India)' },
  { code: 'te-IN', name: 'Telugu (India)' },
  { code: 'th-TH', name: 'Thai (Thailand)' },
  { code: 'tr-TR', name: 'Turkish (Turkey)' },
  { code: 'uk-UA', name: 'Ukrainian (Ukraine)' },
  { code: 'ur-IN', name: 'Urdu (India)' },
  { code: 'vi-VN', name: 'Vietnamese (Vietnam)' },
];

export const GOOGLE_VOICES = [
  { name: 'achernar', description: 'Soft', gender: 'Female' },
  { name: 'achird', description: 'Friendly', gender: 'Male' },
  { name: 'algenib', description: 'Gravelly', gender: 'Male' },
  { name: 'algieba', description: 'Smooth', gender: 'Male' },
  { name: 'alnilam', description: 'Firm', gender: 'Male' },
  { name: 'aoede', description: 'Breezy', gender: 'Female' },
  { name: 'autonoe', description: 'Bright', gender: 'Female' },
  { name: 'callirrhoe', description: 'Easy-going', gender: 'Female' },
  { name: 'charon', description: 'Informative', gender: 'Male' },
  { name: 'despina', description: 'Smooth', gender: 'Female' },
  { name: 'enceladus', description: 'Breathy', gender: 'Male' },
  { name: 'erinome', description: 'Clear', gender: 'Female' },
  { name: 'fenrir', description: 'Excitable', gender: 'Male' },
  { name: 'gacrux', description: 'Mature', gender: 'Female' },
  { name: 'iapetus', description: 'Clear', gender: 'Male' },
  { name: 'kore', description: 'Firm', gender: 'Female' },
  { name: 'laomedeia', description: 'Upbeat', gender: 'Female' },
  { name: 'leda', description: 'Youthful', gender: 'Female' },
  { name: 'orus', description: 'Firm', gender: 'Male' },
  { name: 'pulcherrima', description: 'Forward', gender: 'Female' },
  { name: 'puck', description: 'Upbeat', gender: 'Male' },
  { name: 'rasalgethi', description: 'Informative', gender: 'Male' },
  { name: 'sadachbia', description: 'Lively', gender: 'Male' },
  { name: 'sadaltager', description: 'Knowledgeable', gender: 'Male' },
  { name: 'schedar', description: 'Even', gender: 'Male' },
  { name: 'sulafat', description: 'Warm', gender: 'Female' },
  { name: 'umbriel', description: 'Easy-going', gender: 'Male' },
  { name: 'vindemiatrix', description: 'Gentle', gender: 'Female' },
  { name: 'zephyr', description: 'Bright', gender: 'Female' },
  { name: 'zubenelgenubi', description: 'Casual', gender: 'Male' },
];
export const GOOGLE_MODELS = [
  {
    name: 'Chirp3-HD',
    id: 'Chirp3-HD',
  },
];

// export const GOOGLE_SAMPLE_RATES = [
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
// export const GOOGLE_ENCODINGS = [
//   {
//     name: 'MULAW',
//     id: 'MULAW',
//     value: 'mulaw',
//   },
//   {
//     name: 'ALAW',
//     id: 'ALAW',
//     value: 'alaw',
//   },
//   {
//     name: 'PCM',
//     id: 'PCM',
//     value: 'pcm',
//   },
// ];

export const GetGoogleDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];
  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'speak.language',
    'speak.voice.id',
    'speak.model',
    'speak.output_format.sample_rate',
    'speak.output_format.encoding',
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
  addMetadata('speak.language', 'en-US', value =>
    GOOGLE_LANGUAGE.some(l => l.code === value),
  );

  // Set voice
  addMetadata('speak.voice.id', 'achernar', value =>
    GOOGLE_VOICES.some(v => v.name === value),
  );

  // Set model
  addMetadata('speak.model', 'Chirp3-HD', value =>
    GOOGLE_MODELS.some(m => m.id === value),
  );

  // Only return metadata for the keys we want to keep
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('speaker.')),
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
  const languageOption = options.find(opt => opt.getKey() === 'speak.language');
  if (
    !languageOption ||
    !GOOGLE_LANGUAGE.some(lang => lang.code === languageOption.getValue())
  ) {
    return false;
  }

  // Validate voice
  const voiceOption = options.find(opt => opt.getKey() === 'speak.voice.id');
  if (
    !voiceOption ||
    !GOOGLE_VOICES.some(voice => voice.name === voiceOption.getValue())
  ) {
    return false;
  }

  // Validate model
  const modelOption = options.find(opt => opt.getKey() === 'speak.model');
  if (
    !modelOption ||
    !GOOGLE_MODELS.some(model => model.id === modelOption.getValue())
  ) {
    return false;
  }

  return true;
};
