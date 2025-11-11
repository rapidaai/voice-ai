import { Metadata } from '@rapidaai/react';

export interface ProviderConfig {
  providerId: string;
  provider: string;
  parameters: Metadata[];
}

export type IntegrationProvider = {
  id: string;
  code: string;
  name: string;
  description: string;
  image: string;
  featureList: string[];
  configurations: {
    name: string;
    type: 'string' | 'text' | 'select'; // Type now includes 'select'
    label: string;
    options?: string[]; // Opti
  }[];
};

export interface RapidaProvider extends IntegrationProvider {
  humanname?: string;
  website?: string;
  status?: string;
}

export const COMPLETE_PROVIDER: RapidaProvider[] = [
  //
  // telephony
  {
    id: '7646645603519189353',
    code: 'vonage',
    name: 'Vonage',
    description:
      'Cloud communication platform that provides voice, messaging, and video solutions for businesses.',
    image: 'https://cdn-01.rapida.ai/partners/vonage.jpeg',
    featureList: ['telephony', 'external'],
    configurations: [
      { name: 'application_id', type: 'string', label: 'Application Id' },
      { name: 'private_key', type: 'text', label: 'Private key' },
    ],
  },
  {
    id: '7646645603519189354',
    name: 'Exotel',
    code: 'exotel',
    description:
      'Cloud telephony platform offering voice and messaging services for businesses.',
    image: 'https://cdn-01.rapida.ai/partners/exotel.jpeg',
    featureList: ['telephony', 'external'],
    configurations: [
      { name: 'account_sid', type: 'string', label: 'Account sid' },
      { name: 'client_id', type: 'string', label: 'Client id' },
      { name: 'client_secret', type: 'string', label: 'Client secret' },
    ],
  },

  {
    id: '7835356314600149384',
    name: 'Twilio',
    code: 'twilio',
    description:
      'Cloud communications platform for building SMS, voice, and messaging applications',
    image: 'https://cdn-01.rapida.ai/partners/tools/twilio-icon.svg',
    featureList: ['telephony', 'external'],
    configurations: [
      { name: 'account_token', type: 'string', label: 'Account token' },
      { name: 'account_sid', type: 'string', label: 'Account sid' },
    ],
  },

  // cloud provider
  {
    id: '1254821443460603620',
    code: 'aws-cloud',
    name: 'AWS',
    description:
      'Amazon Web Services (AWS) offers a broad set of global cloud-based products including compute, storage, databases, analytics, and AI services.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/1254821443460603620.png',
    featureList: ['storage'],
    configurations: [
      {
        name: 'access_key_id',
        type: 'string',
        label: 'access_key_id',
      },
      {
        name: 'secret_access_key',
        type: 'string',
        label: 'secret_access_key',
      },
      {
        name: 'region',
        type: 'string',
        label: 'region',
      },
    ],
    humanname: 'AWS',
    website: 'https://aws.amazon.com/bedrock/',
    status: 'ACTIVE',
  },
  {
    id: '125482144346060362',
    code: 'google-cloud',
    name: 'Google Cloud',
    description:
      'Google Cloud offers a suite of cloud computing services that runs on the same infrastructure that Google uses internally for its end-user products.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742118.jpg',
    featureList: ['storage', 'stt', 'tts'],
    configurations: [
      {
        name: 'project_id',
        type: 'string',
        label: 'Project ID',
      },
      {
        name: 'service_account_key',
        type: 'text',
        label: 'Service Account Key (JSON)',
      },
    ],
    humanname: 'Google Cloud',
    website: 'https://cloud.google.com/',
    status: 'ACTIVE',
  },
  //   {
  //     id: '12548214434606036',
  //     code: 'azure-cloud',
  //     name: 'Azure Cloud',
  //     description:
  //       'Microsoft Azure is a cloud computing platform and infrastructure for building, deploying, and managing applications and services through Microsoft-managed data centers.',
  //     image:
  //       'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742122.png',
  //     featureList: ['storage'],
  //     configurations: [
  //       {
  //         name: 'subscription_id',
  //         type: 'string',
  //         label: 'Subscription ID',
  //       },
  //       {
  //         name: 'tenant_id',
  //         type: 'string',
  //         label: 'Tenant ID',
  //       },
  //       {
  //         name: 'client_id',
  //         type: 'string',
  //         label: 'Client ID',
  //       },
  //       {
  //         name: 'client_secret',
  //         type: 'string',
  //         label: 'Client Secret',
  //       },
  //     ],
  //     humanname: 'Azure Cloud',
  //     website: 'https://azure.microsoft.com/',
  //     status: 'ACTIVE',
  //   },

  //   models
  {
    id: '198796716894742122',
    code: 'azure',
    name: 'Azure',
    description:
      'Azure AI Foundry Service enables to access all the cognitive apis like openai, stt, tts',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742122.png',
    featureList: ['text', 'stt', 'tts'],
    configurations: [
      {
        name: 'subscription_key',
        type: 'string',
        label: 'Subscription Key',
      },
      {
        name: 'endpoint',
        type: 'string',
        label: 'Endpoint',
      },
    ],
    humanname: 'Azure OpenAI Service',
    website:
      'https://azure.microsoft.com/en-us/products/ai-services/openai-service',
    status: 'ACTIVE',
  },
  {
    id: '1987967168347635712',
    code: 'anthropic',
    name: 'Anthropic',
    description:
      'A faster, cheaper yet still very capable version of Claude, which can handle a range of tasks including casual dialogue, text analysis, summarization, and document comprehension.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/1987967168347635712.jpg',
    featureList: ['text'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Anthropic',
    website: 'https://www.anthropic.com',
    status: 'ACTIVE',
  },
  {
    id: '1987967168435716096',
    code: 'cohere',
    name: 'Cohere',
    description:
      "A smaller and faster version of Cohere's command model with almost as much capability but improved speed.",
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/1987967168435716096.png',
    featureList: ['text', 'embedding'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Cohere',
    website: 'https://cohere.com',
    status: 'ACTIVE',
  },
  {
    id: '1987967168452493312',
    code: 'openai',
    name: 'OpenAI',
    description:
      'GPT-4 from OpenAI has broad general knowledge and domain expertise allowing it to follow complex instructions in natural language and solve difficult problems accurately.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/1987967168452493312.svg',
    featureList: ['text', 'embedding'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'OpenAI',
    website: 'https://openai.com',
    status: 'ACTIVE',
  },
  {
    id: '198796716894742119',
    code: 'mistral',
    name: 'Mistral',
    description:
      'Mistral specializes in creating fast, secure, open-source large language models (LLMs).',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742119.jpeg',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Mistral',
    website: 'https://mistral.ai',
    status: 'ACTIVE',
  },
  {
    id: '198796716894742120',
    code: 'huggingface',
    name: 'Hugging Face',
    description:
      'Hugging Face is a machine learning (ML) and data science platform and community that helps users build, deploy and train machine learning models.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742120.jpeg',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Hugging Face',
    website: 'https://huggingface.co',
    status: 'ACTIVE',
  },
  {
    id: '198796716894742118',
    code: 'google',
    name: 'Google',
    description: 'Gemini is Google latest family of large language models.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742118.jpg',
    featureList: ['text', 'stt', 'tts', 'embedding'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Google',
    website: 'https://ai.google.dev/',
    status: 'ACTIVE',
  },
  {
    id: '1987967168431521792',
    code: 'replicate',
    name: 'Replicate',
    description:
      '7 billion parameter open source model by Meta fine-tuned for chat purposes served by Fireworks. LLaMA v2 was trained on more data (~2 trillion tokens) compared to LLaMA v1 and supports context windows up to 4k tokens.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/1987967168431521792.jpg',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Replicate',
    website: 'https://ai.meta.com/llama/',
    status: 'ACTIVE',
  },
  {
    id: '198796716894742121',
    code: 'stabilityai',
    name: 'Stability AI',
    description:
      'Stability AI is a developer of an open AI model for images, languages, audio, video, 3D, and biology. These open AI tools are created to help developers design and implement solutions through collective intelligence and augmented technology.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742121.jpg',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Stability AI',
    website: 'https://platform.stability.ai',
    status: 'ACTIVE',
  },
  {
    id: '198796716894742124',
    code: 'ai21',
    name: 'A21 labs',
    description:
      'The A21 Campaign is a global 501 non-profit, non-governmental organization that works to fight human trafficking, including sexual exploitation and trafficking, forced slave labor, bonded labor, involuntary domestic servitude, and child soldiery.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742124.jpg',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'A21 labs',
    website: 'https://www.ai21.com/',
    status: 'ACTIVE',
  },
  {
    id: '198796716894742125',
    code: 'deepinfra',
    name: 'deepinfra',
    description:
      'Deep Infra offers cost-effective, scalable, easy-to-deploy, and production-ready machine-learning models and infrastructures for deep-learning models.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742125.jpg',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'deepinfra',
    website: 'https://deepinfra.com/',
    status: 'ACTIVE',
  },
  {
    id: '198796716894742126',
    code: 'pplx-api',
    name: 'perplexity ai',
    description:
      'Perplexity AI is a conversational search engine that provides answers to queries using natural language predictive text.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742126.jpg',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'perplexity ai',
    website: 'https://docs.perplexity.ai/',
    status: 'ACTIVE',
  },
  {
    id: '198796716894742123',
    code: 'togetherai',
    name: 'Together AI',
    description:
      'Together AI is a research-driven artificial intelligence company. We contribute leading open-source research, models, and datasets to advance the frontier of AI. Our decentralized cloud services empower developers and researchers at organizations of all sizes to train, fine-tune, and deploy generative AI models.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742123.png',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Together AI',
    website: 'https://www.together.ai/',
    status: 'ACTIVE',
  },
  {
    id: '5212367370329048775',
    code: 'voyageai',
    name: 'VoyageAI',
    description:
      'VoyageAI, a cutting-edge AI platform offering comprehensive solutions for various AI applications with high efficiency and performance.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/5212367370329048775.png',
    featureList: ['embedding'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'VoyageAI',
    website: 'https://voyageai.com',
    status: 'ACTIVE',
  },
  {
    id: '8298870085084815298',
    code: 'rapida',
    name: 'Rapida',
    description: 'Rapida is a provider of advanced generative ai router.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/rapida.png',
    featureList: [],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Rapida',
    website: 'https://rapida.ai',
    status: 'ACTIVE',
  },
  {
    id: '2123891723608588082',
    code: 'deepgram',
    name: 'Deepgram',
    description:
      'Deepgram provides cutting-edge speech recognition services using advanced AI technology. Known for high accuracy and speed in transcribing spoken language into text.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/2123891723608588082.jpg',
    featureList: ['stt', 'tts'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'Deepgram',
    website: 'https://deepgram.com',
    status: 'ACTIVE',
  },
  {
    id: '21238917236010',
    code: 'cartesia',
    name: 'Cartesia',
    description: 'Advanced Text-to-Speech and Speech-to-Text solutions',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/cartesia.jpg',
    featureList: ['stt', 'tts'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'cartesia',
    website: 'https://cartesia.com',
    status: 'ACTIVE',
  },
  {
    id: '21238917236011',
    code: 'elevenlabs',
    name: 'ElevenLabs',
    description:
      'ElevenLabs offers state-of-the-art AI voice technology, providing high-quality text-to-speech synthesis with natural-sounding voices and customization options.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/11labs.png', // You'll need to update this URL
    featureList: ['tts'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'ElevenLabs',
    website: 'https://elevenlabs.io',
    status: 'ACTIVE',
  },
  {
    id: '21238917236012', // You may want to change this ID
    code: 'assemblyai',
    name: 'AssemblyAI',
    description:
      'AssemblyAI offers state-of-the-art speech recognition and audio intelligence APIs. Known for high accuracy and advanced features like speaker diarization and sentiment analysis.',
    image:
      'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/assemblyai.png', // Update this URL to AssemblyAI's logo
    featureList: ['stt'],
    configurations: [
      {
        name: 'key',
        type: 'string',
        label: 'API Key',
      },
    ],
    humanname: 'AssemblyAI',
    website: 'https://www.assemblyai.com',
    status: 'ACTIVE',
  },
];

export const TEXT_PROVIDERS = COMPLETE_PROVIDER.filter(x =>
  x.featureList.includes('text'),
);
export const INTEGRATION_PROVIDER: IntegrationProvider[] =
  COMPLETE_PROVIDER.filter(x => x.featureList.includes('external'));
export const SPEECH_TO_TEXT_PROVIDER = COMPLETE_PROVIDER.filter(x =>
  x.featureList.includes('stt'),
);
export const TEXT_TO_SPEECH_PROVIDER = COMPLETE_PROVIDER.filter(x =>
  x.featureList.includes('tts'),
);

export const STORAGE_PROVIDER = COMPLETE_PROVIDER.filter(x =>
  x.featureList.includes('storage'),
);
export const TELEPHONY_PROVIDER = COMPLETE_PROVIDER.filter(x =>
  x.featureList.includes('telephony'),
);

export const EMBEDDING_PROVIDERS = COMPLETE_PROVIDER.filter(x =>
  x.featureList.includes('embedding'),
);

// export const EMBEDDING_PROVIDERS = [
//   {
//     id: '1987967168435716096',
//     created_date: '2023-11-18 22:21:47.599924',
//     updated_date: null,
//     name: 'cohere',
//     description:
//       "A smaller and faster version of Cohere's command model with almost as much capability but improved speed.",
//     human_name: 'Cohere',
//     website: 'https://cohere.com',
//     image:
//       'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/1987967168435716096.png',
//     status: 'ACTIVE',
//     connect_configuration: [
//       { name: 'key', type: 'string', label: 'Provider Key' },
//     ],
//   },
//   {
//     id: '1987967168452493312',
//     created_date: '2023-11-18 22:21:47.60321',
//     updated_date: null,
//     name: 'openai',
//     description:
//       'GPT-4 from OpenAI has broad general knowledge and domain expertise allowing it to follow complex instructions in natural language and solve difficult problems accurately.',
//     human_name: 'OpenAI',
//     website: 'https://openai.com',
//     image:
//       'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/1987967168452493312.svg',
//     status: 'ACTIVE',
//     connect_configuration: [
//       { name: 'key', type: 'string', label: 'Provider Key' },
//     ],
//   },
//   {
//     id: '198796716894742118',
//     created_date: '2023-12-13 17:18:53.32083',
//     updated_date: null,
//     name: 'google',
//     description: 'Gemini is Google latest family of large language models.',
//     human_name: 'Google',
//     website: 'https://ai.google.dev/',
//     image:
//       'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/198796716894742118.jpg',
//     status: 'ACTIVE',
//     connect_configuration: [
//       { name: 'key', type: 'string', label: 'Provider Key' },
//     ],
//   },
//   {
//     id: '5212367370329048775',
//     created_date: '2024-07-08 05:56:08.22595',
//     updated_date: null,
//     name: 'voyageai',
//     description:
//       'VoyageAI, a cutting-edge AI platform offering comprehensive solutions for various AI applications with high efficiency and performance.',
//     human_name: 'VoyageAI',
//     website: 'https://voyageai.com',
//     image:
//       'https://rapida-assets-01.s3.ap-south-1.amazonaws.com/providers/5212367370329048775.png',
//     status: 'ACTIVE',
//     connect_configuration: [
//       { name: 'key', type: 'string', label: 'Provider Key' },
//     ],
//   },
// ];
