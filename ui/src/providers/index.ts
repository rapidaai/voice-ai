import productionProvider from './provider.production.json';
import developmentProvider from './provider.development.json';

export interface IntegrationProvider extends RapidaProvider { }
interface EndOfSpeechProvider extends RapidaProvider { }
interface VADProvider extends RapidaProvider { }
interface NoiseCancellationProvider extends RapidaProvider { }
export interface RapidaProviderConfiguration {
  name: string;
  type: string;
  label: string;
  required?: boolean;
  choices?: { label: string; value: string }[];
  /** Example value shown inside the empty field. */
  placeholder?: string;
  /** Short description rendered under the field. */
  helperText?: string;
  /** Renders as a password input; never read back from the API. */
  secret?: boolean;
  /** Grouped under the collapsed "advanced settings" section. */
  advanced?: boolean;
}

export interface RapidaProvider {
  code: string;
  name: string;
  featureList: string[];
  description?: string;
  image?: string;
  url?: string;
  configurations?: RapidaProviderConfiguration[];
}

/**
 *
 * @returns
 */

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

export const TELEMETRY_PROVIDER = allProvider().filter(x =>
  x.featureList.includes('telemetry'),
);

/**
 *
 * Azure speech service constants
 * @returns
 */

export const AZURE_TEXT_TO_SPEECH_VOICE = () => {
  return require('./azure-speech-service/voices.json');
};

export const AZURE_TEXT_TO_SPEECH_LANGUAGE = () => {
  return require('./azure-speech-service/text-to-speech-language.json');
};

export const AZURE_SPEECH_TO_TEXT_LANGUAGE = () => {
  return require('./azure-speech-service/speech-to-text-language.json');
};

/**
 *
 * @returns
 */
export const GOOGLE_CLOUD_VOICE = () => {
  return require('./google/interim.voice.json');
};
export const GOOGLE_SPEECH_TO_TEXT_MODEL = () => {
  return require('./google/speech-to-text-model.json');
};
export const GOOGLE_SPEECH_TO_TEXT_LANGUGAE = () => {
  return require('./google/speech-to-text-language.json');
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
 * Rime
 */
export const RIME_MODEL = () => {
  return require('./rime/models.json');
};

export const RIME_VOICE = (model?: string, language?: string) => {
  const voices = require('./rime/voices.json');
  if (model && language && voices[model]?.[language]) {
    return voices[model][language];
  }
  if (model && voices[model]) {
    return Object.values(voices[model]).flat();
  }
  return Object.values(voices).flatMap((m: any) => Object.values(m).flat());
};

export const RIME_LANGUAGE = (model?: string) => {
  const allLanguages = require('./rime/languages.json');
  if (model) {
    const voices = require('./rime/voices.json');
    const modelVoices = voices[model];
    if (modelVoices) {
      const availableLangIds = Object.keys(modelVoices);
      return allLanguages.filter((l: any) =>
        availableLangIds.includes(l.language_id),
      );
    }
    return [];
  }
  return allLanguages;
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

export const GEMINI_EMBEDDING_MODEL = () => {
  return require('./gemini/text-embedding-models.json');
};

/**
 *
 */
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

export const SARVAM_VOICE = (model?: string) => {
  const voices = require('./sarvam/voices.json');
  if (model && voices[model]) {
    return voices[model];
  }
  // Default: return all voices flattened
  return Object.values(voices).flat();
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
 * Resemble AI
 */
export const RESEMBLEAI_VOICE = () => {
  return require('./resembleai/voices.json');
};

/**
 * Neuphonic
 */
export const NEUPHONIC_VOICE = () => {
  return require('./neuphonic/voices.json');
};

export const NEUPHONIC_LANGUAGE = () => {
  return require('./neuphonic/languages.json');
};

/**
 * MiniMax
 */
export const MINIMAX_MODEL = () => {
  return require('./minimax/models.json');
};

export const MINIMAX_VOICE = () => {
  return require('./minimax/voices.json');
};

/**
 * Groq
 */
export const GROQ_SPEECH_TO_TEXT_MODEL = () => {
  return require('./groq/speech-to-text-models.json');
};

export const GROQ_SPEECH_TO_TEXT_LANGUAGE = () => {
  return require('./groq/speech-to-text-languages.json');
};

export const GROQ_TEXT_TO_SPEECH_MODEL = () => {
  return require('./groq/text-to-speech-models.json');
};

export const GROQ_VOICE = () => {
  return require('./groq/voices.json');
};

/**
 * Speechmatics
 */
export const SPEECHMATICS_LANGUAGE = () => {
  return require('./speechmatics/languages.json');
};

export const SPEECHMATICS_VOICE = () => {
  return require('./speechmatics/voices.json');
};

/**
 * NVIDIA
 */
export const NVIDIA_LANGUAGE = () => {
  return require('./nvidia/languages.json');
};

export const NVIDIA_VOICE = () => {
  return require('./nvidia/voices.json');
};

/**
 * AWS
 */
export const AWS_SPEECH_TO_TEXT_LANGUAGE = () => {
  return require('./aws/speech-to-text-languages.json');
};

export const AWS_TEXT_TO_SPEECH_MODEL = () => {
  return require('./aws/text-to-speech-models.json');
};

export const AWS_VOICE = () => {
  return require('./aws/voices.json');
};

export const AWS_LANGUAGE = () => {
  return require('./aws/languages.json');
};
