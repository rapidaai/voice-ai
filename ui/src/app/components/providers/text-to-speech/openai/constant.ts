import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const OPENAI_VOICES = [
  {
    name: 'Alloy',
    id: 'alloy',
    description: 'A robust and clear voice.',
    characteristics: ['Clear', 'Robust'],
  },
  {
    name: 'Echo',
    id: 'echo',
    description: 'A warm and inviting voice.',
    characteristics: ['Warm', 'Inviting'],
  },
  {
    name: 'Fable',
    id: 'fable',
    description: 'A smooth and adaptable voice.',
    characteristics: ['Smooth', 'Adaptable'],
  },
  {
    name: 'Onyx',
    id: 'onyx',
    description: 'A deep and commanding voice.',
    characteristics: ['Deep', 'Commanding'],
  },
  {
    name: 'Nova',
    id: 'nova',
    description: 'A clear and expressive female voice.',
    characteristics: ['Clear', 'Expressive', 'Female'],
  },
  {
    name: 'Shimmer',
    id: 'shimmer',
    description: 'A bright and cheerful female voice.',
    characteristics: ['Bright', 'Cheerful', 'Female'],
  },
];

export const OPENAI_MODELS = [
  {
    name: 'TTS-1',
    id: 'tts-1',
    description:
      "OpenAI's standard latency text-to-speech model, designed for quick audio generation.",
    quality_level: 'Standard Latency',
    intended_use_cases: [
      'Real-time conversational AI',
      'Interactive voice responses',
      'Applications requiring rapid audio output',
    ],
    key_characteristics: [
      'Optimized for speed and minimal delay.',
      'Good balance of quality and performance for interactive scenarios.',
    ],
  },
  {
    name: 'TTS-1-HD',
    id: 'tts-1-hd',
    description:
      "OpenAI's high-definition text-to-speech model, focused on superior audio fidelity.",
    quality_level: 'High Quality (HD)',
    intended_use_cases: [
      'Professional narration (audiobooks, podcasts)',
      'High-quality voiceovers for multimedia content',
      'Any scenario where premium audio fidelity is critical',
    ],
    key_characteristics: [
      'Produces richer, more natural-sounding speech.',
      'May have slightly higher latency compared to `tts-1`.',
      'Minimizes audio artifacts for a cleaner output.',
    ],
  },
  {
    name: 'TTS-1-HD',
    id: 'tts-1-hd',
    description:
      "OpenAI's high-definition text-to-speech model, focused on superior audio fidelity.",
    quality_level: 'High Quality (HD)',
    intended_use_cases: [
      'Professional narration (audiobooks, podcasts)',
      'High-quality voiceovers for multimedia content',
      'Any scenario where premium audio fidelity is critical',
    ],
    key_characteristics: [
      'Produces richer, more natural-sounding speech.',
      'May have slightly higher latency compared to `tts-1`.',
      'Minimizes audio artifacts for a cleaner output.',
    ],
  },
  {
    name: 'GPT-4O-Mini-TTS',
    id: 'gpt-4o-mini-tts',
    description:
      "OpenAI's compact text-to-speech model, optimized for efficiency and speed.",
    quality_level: 'Standard Quality',
    intended_use_cases: [
      'Quick voice responses in chatbots or virtual assistants',
      'Rapid prototyping of voice applications',
      'Scenarios where low latency is prioritized over audio fidelity',
    ],
    key_characteristics: [
      'Faster processing and lower latency compared to larger models.',
      'Smaller model size, suitable for edge devices or constrained environments.',
      'Balances quality and performance for efficient text-to-speech conversion.',
    ],
  },
];

export const OPENAI_LANGUAGES = [
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

export const GetOpenAIDefaultOptions = (current: Metadata[]): Metadata[] => {
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
    OPENAI_LANGUAGES.some(l => l.code === value),
  );

  // Set voice
  addMetadata('speak.voice.id', 'alloy', value =>
    OPENAI_VOICES.some(v => v.id === value),
  );

  // Set model
  addMetadata('speak.model', 'gpt-4o-mini-tts', value =>
    OPENAI_MODELS.some(m => m.id === value),
  );

  // Only return metadata for the keys we want to keep
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('speaker.')),
  ];
};

export const ValidateOpenAIOptions = (options: Metadata[]): boolean => {
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
    !OPENAI_LANGUAGES.some(lang => lang.code === languageOption.getValue())
  ) {
    return false;
  }

  // Validate voice
  const voiceOption = options.find(opt => opt.getKey() === 'speak.voice.id');
  if (
    !voiceOption ||
    !OPENAI_VOICES.some(voice => voice.id === voiceOption.getValue())
  ) {
    return false;
  }

  // Validate model
  const modelOption = options.find(opt => opt.getKey() === 'speak.model');
  if (
    !modelOption ||
    !OPENAI_MODELS.some(model => model.id === modelOption.getValue())
  ) {
    return false;
  }

  return true;
};
