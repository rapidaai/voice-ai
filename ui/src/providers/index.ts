import productionProvider from './provider.production.json';
import developmentProvider from './provider.development.json';

export interface IntegrationProvider extends RapidaProvider {}
interface EndOfSpeechProvider extends RapidaProvider {}
interface VADProvider extends RapidaProvider {}
interface NoiseCancellationProvider extends RapidaProvider {}
export interface RapidaProvider {
  code: string;
  name: string;
  featureList: string[];
  description?: string;
  image?: string;
  url?: string;
  configurations?: {
    name: string;
    type: string;
    label: string;
  }[];
}

/**
 *
 * @returns
 */

export const SENTENCE_BOUNDRIES = [
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
];

export const PRONUNCIATION_DICTIONARIES = [
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
];

export const CONJUNCTION_BOUNDARIES = [
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
];

export const allProvider = (): RapidaProvider[] => {
  const env = process.env.NODE_ENV || 'development';
  return env === 'production' ? productionProvider : developmentProvider;
};

export const EndOfSpeech = (): EndOfSpeechProvider[] => {
  return allProvider().filter(x => x.featureList.includes('end_of_speech'));
};

export const VAD = (): VADProvider[] => {
  return allProvider().filter(x => x.featureList.includes('vad'));
};

export const NoiseCancellation = (): NoiseCancellationProvider[] => {
  return allProvider().filter(x =>
    x.featureList.includes('noise_cancellation'),
  );
};

export const TEXT_PROVIDERS = allProvider().filter(x =>
  x.featureList.includes('text'),
);
export const INTEGRATION_PROVIDER: IntegrationProvider[] = allProvider().filter(
  x => x.featureList.includes('external'),
);
export const SPEECH_TO_TEXT_PROVIDER = allProvider().filter(x =>
  x.featureList.includes('stt'),
);
export const TEXT_TO_SPEECH_PROVIDER = allProvider().filter(x =>
  x.featureList.includes('tts'),
);
export const TEXT_TO_SPEECH = (name: string) =>
  allProvider()
    .filter(x => x.featureList.includes('tts'))
    .findLast(x => x.code === name);

export const STORAGE_PROVIDER = allProvider().filter(x =>
  x.featureList.includes('storage'),
);

export const RERANKER_PROVIDER = allProvider().filter(x =>
  x.featureList.includes('reranker'),
);

export const TELEPHONY_PROVIDER = allProvider().filter(x =>
  x.featureList.includes('telephony'),
);

export const EMBEDDING_PROVIDERS = allProvider().filter(x =>
  x.featureList.includes('embedding'),
);

/**
 *
 * @returns
 */

export const AZURE_VOICE = () => {
  return require('./azure/voices.json');
};

export const AZURE_LANGUAGE = () => {
  return require('./azure/languages.json');
};

export const AZURE_TEXT_MODEL = () => {
  return require('./azure/text-models.json');
};

/**
 *
 * @returns
 */
export const GOOGLE_CLOUD_VOICE = () => {
  return require('./google/interim.voice.json');
};

/**
 *
 * @returns
 */
export const DEEPGRAM_VOICE = () => {
  return require('./deepgram/voices.json');
};

export const DEEPGRAM_SPEECH_TO_TEXT_MODEL = () => {
  return require('./deepgram/speech-to-text-models.json');
};

export const DEEPGRAM_SPEECH_TO_TEXT_LANGUAGE = () => {
  return require('./deepgram/speech-to-text-languages.json');
};

/**
 * ElevEnlabs constants
 * @returns
 */
export const ELEVENLABS_MODEL = () => {
  return require('./elevenlabs/models.json');
};

export const ELEVENLABS_VOICE = () => {
  return require('./elevenlabs/voices.json');
};

export const ELEVENLABS_LANGUAGE = () => {
  return require('./elevenlabs/languages.json');
};

/**
 * cartesia
 */

//

export const CARTESIA_VOICE = () => {
  return require('./cartesia/voices.json');
};

export const CARTESIA_TEXT_TO_SPEECH_MODEL = () => {
  return require('./cartesia/text-to-speech-models.json');
};

export const CARTESIA_SPEECH_TO_TEXT_MODEL = () => {
  return require('./cartesia/speech-to-text-models.json');
};

export const CARTESIA_LANGUAGE = () => {
  return require('./cartesia/languages.json');
};

export const CARTESIA_SPEED_OPTION = () => {
  return [
    { id: '', name: '' },
    { id: 'slowest', name: 'Slowest' },
    { id: 'slow', name: 'Slow' },
    { id: 'normal', name: 'Normal' },
    { id: 'fast', name: 'Fast' },
    { id: 'fastest', name: 'Fastest' },
  ];
};

export const CARTESIA_EMOTION_LEVEL_COMBINATION = [
  ...[
    { id: 'anger', name: 'Anger' },
    { id: 'positivity', name: 'Positivity' },
    { id: 'surprise', name: 'Surprise' },
    { id: 'sadness', name: 'Sadness' },
    { id: 'curiosity', name: 'Curiosity' },
  ].flatMap(m => m.id),
  ...[
    { id: 'anger', name: 'Anger' },
    { id: 'positivity', name: 'Positivity' },
    { id: 'surprise', name: 'Surprise' },
    { id: 'sadness', name: 'Sadness' },
    { id: 'curiosity', name: 'Curiosity' },
  ].flatMap(emotion =>
    [
      { id: 'lowest', name: 'Lowest' },
      { id: 'low', name: 'Low' },
      { id: 'high', name: 'High' },
      { id: 'highest', name: 'Highest' },
    ].map(level => `${emotion.id}:${level.id}`),
  ),
];

/**
 *
 */

export const GEMINI_MODEL = () => {
  return require('./gemini/models.json');
};

/**
 * sarvam
 */

export const SARVAM_LANGUAGE = () => {
  return require('./sarvam/languages.json');
};

export const SARVAM_TEXT_TO_SPEECH_MODEL = () => {
  return require('./sarvam/text-to-speech-models.json');
};

export const SARVAM_SPEECH_TO_TEXT_MODEL = () => {
  return require('./sarvam/speech-to-text-models.json');
};

export const SARVAM_VOICE = () => {
  return require('./sarvam/voices.json');
};

/**
 * assembly
 */

export const ASSEMBLYAI_SPEECH_TO_TEXT_MODEL = () => {
  return require('./assemblyai/speech-to-text-models.json');
};
export const ASSEMBLYAI_LANGUAGE = () => {
  return require('./assemblyai/languages.json');
};

/**
 *
 */
