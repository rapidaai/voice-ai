import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const DEEPGRAM_VOICES = [
  {
    name: 'Thalia',
    id: 'thalia',
    language_code: 'en',
    accent: 'American',
    expressed_gender: 'feminine',
    characteristics: ['Clear', 'Confident', 'Energetic', 'Enthusiastic'],
  },
  {
    name: 'Zeus',
    id: 'zeus',
    language_code: 'en',
    accent: 'American',
    expressed_gender: 'masculine',
    characteristics: ['Deep', 'Trustworthy', 'Smooth'],
  },
  {
    name: 'Andromeda',
    id: 'andromeda',
    language_code: 'en',
    accent: 'American',
    expressed_gender: 'feminine',
    characteristics: ['Casual', 'Expressive', 'Comfortable'],
  },
  {
    name: 'Apollo',
    id: 'apollo',
    language_code: 'en',
    accent: 'American',
    expressed_gender: 'masculine',
    characteristics: ['Confident', 'Comfortable', 'Casual'],
  },
  {
    name: 'Athena',
    id: 'athena',
    language_code: 'en',
    accent: 'American',
    expressed_gender: 'feminine',
    characteristics: ['Calm', 'Smooth', 'Professional'],
  },
  {
    name: 'Draco',
    id: 'draco',
    language_code: 'en',
    accent: 'British',
    expressed_gender: 'masculine',
    characteristics: ['Natural', 'Warm', 'Professional'],
  },
  {
    name: 'Selene',
    id: 'selene',
    language_code: 'en',
    accent: 'American',
    expressed_gender: 'feminine',
    characteristics: ['Expressive', 'Engaging', 'Energetic'],
  },
  {
    name: 'Hyperion',
    id: 'hyperion',
    language_code: 'en',
    accent: 'Australian',
    expressed_gender: 'masculine',
    characteristics: ['Caring', 'Warm', 'Empathetic'],
  },
  {
    name: 'Theia',
    id: 'theia',
    language_code: 'en',
    accent: 'Australian',
    expressed_gender: 'feminine',
    characteristics: ['Expressive', 'Polite', 'Sincere'],
  },
  {
    name: 'Vesta',
    id: 'vesta',
    language_code: 'en',
    accent: 'American',
    expressed_gender: 'feminine',
    characteristics: ['Natural', 'Expressive', 'Patient', 'Empathetic'],
  },
  {
    name: 'Celeste',
    id: 'celeste',
    language_code: 'es',
    accent: 'Latin American (General)',
    expressed_gender: 'feminine',
    characteristics: ['Warm', 'Engaging', 'Clear'],
  },
  {
    name: 'Eros',
    id: 'eros',
    language_code: 'es',
    accent: 'Spanish (Castilian)',
    expressed_gender: 'masculine',
    characteristics: ['Formal', 'Clear', 'Confident'],
  },
];

export const DEEPGRAM_MODELS = [
  {
    name: 'Aura-2',
    id: 'aura-2',
    description:
      "Deepgram's next-generation text-to-speech API, engineered to deliver natural, professional speech with real-time performance and domain-specific accuracy.",
    quality_level: 'High Quality, Real-time Performance',
    intended_use_cases: [
      'Real-time AI agents and conversational AI',
      'Customer service automation',
      'IVR (Interactive Voice Response) systems',
      'Applications requiring sub-200ms latency',
    ],
    key_characteristics: [
      'Delivers human-like speech with natural tone, rhythm, and emotion.',
      'Sub-200ms latency for ultra-responsive interactions.',
      'Features 40+ English voices with localized accents (American, British, Australian, Irish, Filipino).',
      'Supports multiple languages including English and Spanish (with various accents).',
      'Context-aware delivery, adjusting pacing and tone.',
      'Scalable infrastructure for high throughput.',
      'Pronunciation accuracy for industry-specific terminology.',
    ],
    notes:
      'When making an API call, you select a specific voice which inherently selects the Aura-2 model. The model ID combines the model version and the voice/language, e.g., `aura-2-thalia-en`.',
  },
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
  { code: 'en', name: 'English' },
  { code: 'es', name: 'Spanish' },
  { code: 'fr', name: 'French' },
  { code: 'de', name: 'German' },
  { code: 'hi', name: 'Hindi' },
  { code: 'ru', name: 'Russian' },
  { code: 'pt', name: 'Portuguese' },
  { code: 'ja', name: 'Japanese' },
  { code: 'it', name: 'Italian' },
  { code: 'nl', name: 'Dutch' },
];

export const GetDeepgramDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];
  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'speak.language',
    'speak.voice.id',
    'speak.model',
  ];
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
  addMetadata('speak.language', 'en', value =>
    DEEPGRAM_LANGUAGES.some(l => l.code === value),
  );

  // Set voice
  addMetadata('speak.voice.id', 'thalia', value =>
    DEEPGRAM_VOICES.some(v => v.id === value),
  );

  // Set model
  addMetadata('speak.model', 'aura-2', value =>
    DEEPGRAM_MODELS.some(m => m.id === value),
  );

  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('speaker.')),
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
  const validations = [
    { key: 'speak.language', validator: DEEPGRAM_LANGUAGES, field: 'code' },
    { key: 'speak.voice.id', validator: DEEPGRAM_VOICES, field: 'id' },
    { key: 'speak.model', validator: DEEPGRAM_MODELS, field: 'id' },
  ];

  return validations.every(({ key, validator, field }) => {
    const option = options.find(opt => opt.getKey() === key);
    return option && validator.some(item => item[field] === option.getValue());
  });
};
