import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const AZURE_VOICE = [
  // Indian English voices
  {
    name: 'AartiIndic (Female)',
    id: 'en-IN-AartiIndicNeural',
    value: 'Female',
  },
  { name: 'ArjunIndic (Male)', id: 'en-IN-ArjunIndicNeural', value: 'Male' },
  {
    name: 'NeerjaIndic (Female)',
    id: 'en-IN-NeerjaIndicNeural',
    value: 'Female',
  },
  {
    name: 'PrabhatIndic (Male)',
    id: 'en-IN-PrabhatIndicNeural',
    value: 'Male',
  },
  { name: 'Aarav (Male)', id: 'en-IN-AaravNeural', value: 'Male' },
  { name: 'Aashi (Female)', id: 'en-IN-AashiNeural', value: 'Female' },
  { name: 'Aarti (Female)', id: 'en-IN-AartiNeural', value: 'Female' },
  { name: 'Arjun (Male)', id: 'en-IN-ArjunNeural', value: 'Male' },
  { name: 'Ananya (Female)', id: 'en-IN-AnanyaNeural', value: 'Female' },
  { name: 'Kavya (Female)', id: 'en-IN-KavyaNeural', value: 'Female' },
  { name: 'Kunal (Male)', id: 'en-IN-KunalNeural', value: 'Male' },
  { name: 'Neerja (Female)', id: 'en-IN-NeerjaNeural', value: 'Female' },
  { name: 'Prabhat (Male)', id: 'en-IN-PrabhatNeural', value: 'Male' },
  { name: 'Rehaan (Male)', id: 'en-IN-RehaanNeural', value: 'Male' },

  // U.S. English voices
  {
    name: 'AvaMultilingual (Female)',
    id: 'en-US-AvaMultilingualNeural4',
    value: 'Female',
  },
  {
    name: 'AndrewMultilingual (Male)',
    id: 'en-US-AndrewMultilingualNeural4',
    value: 'Male',
  },
  {
    name: 'EmmaMultilingual (Female)',
    id: 'en-US-EmmaMultilingualNeural4',
    value: 'Female',
  },
  {
    name: 'AlloyTurboMultilingual (Male)',
    id: 'en-US-AlloyTurboMultilingualNeural4',
    value: 'Male',
  },
  {
    name: 'EchoTurboMultilingual (Male)',
    id: 'en-US-EchoTurboMultilingualNeural4',
    value: 'Male',
  },
  {
    name: 'FableTurboMultilingual (Neutral)',
    id: 'en-US-FableTurboMultilingualNeural4',
    value: 'Neutral',
  },
  {
    name: 'OnyxTurboMultilingual (Male)',
    id: 'en-US-OnyxTurboMultilingualNeural4',
    value: 'Male',
  },
  {
    name: 'NovaTurboMultilingual (Female)',
    id: 'en-US-NovaTurboMultilingualNeural4',
    value: 'Female',
  },
  {
    name: 'ShimmerTurboMultilingual (Female)',
    id: 'en-US-ShimmerTurboMultilingualNeural4',
    value: 'Female',
  },
  {
    name: 'BrianMultilingual (Male)',
    id: 'en-US-BrianMultilingualNeural4',
    value: 'Male',
  },
  { name: 'Ava (Female)', id: 'en-US-AvaNeural', value: 'Female' },
  { name: 'Andrew (Male)', id: 'en-US-AndrewNeural', value: 'Male' },
  { name: 'Emma (Female)', id: 'en-US-EmmaNeural', value: 'Female' },
  { name: 'Brian (Male)', id: 'en-US-BrianNeural', value: 'Male' },
  { name: 'Jenny (Female)', id: 'en-US-JennyNeural', value: 'Female' },
  { name: 'Guy (Male)', id: 'en-US-GuyNeural', value: 'Male' },
  { name: 'Aria (Female)', id: 'en-US-AriaNeural', value: 'Female' },
  { name: 'Davis (Male)', id: 'en-US-DavisNeural', value: 'Male' },
  { name: 'Jane (Female)', id: 'en-US-JaneNeural', value: 'Female' },
  { name: 'Jason (Male)', id: 'en-US-JasonNeural', value: 'Male' },
  { name: 'Kai (Male)', id: 'en-US-KaiNeural', value: 'Male' },
  { name: 'Luna (Female)', id: 'en-US-LunaNeural', value: 'Female' },
  { name: 'Sara (Female)', id: 'en-US-SaraNeural', value: 'Female' },
  { name: 'Tony (Male)', id: 'en-US-TonyNeural', value: 'Male' },
  { name: 'Nancy (Female)', id: 'en-US-NancyNeural', value: 'Female' },
  {
    name: 'CoraMultilingual (Female)',
    id: 'en-US-CoraMultilingualNeural4',
    value: 'Female',
  },
  {
    name: 'ChristopherMultilingual (Male)',
    id: 'en-US-ChristopherMultilingualNeural4',
    value: 'Male',
  },
  {
    name: 'BrandonMultilingual (Male)',
    id: 'en-US-BrandonMultilingualNeural4',
    value: 'Male',
  },
  { name: 'Amber (Female)', id: 'en-US-AmberNeural', value: 'Female' },
  {
    name: 'Ana (Female, Child)',
    id: 'en-US-AnaNeural',
    value: 'Female, Child',
  },
  { name: 'Ashley (Female)', id: 'en-US-AshleyNeural', value: 'Female' },
  { name: 'Brandon (Male)', id: 'en-US-BrandonNeural', value: 'Male' },
  { name: 'Christopher (Male)', id: 'en-US-ChristopherNeural', value: 'Male' },
  { name: 'Cora (Female)', id: 'en-US-CoraNeural', value: 'Female' },
  { name: 'Elizabeth (Female)', id: 'en-US-ElizabethNeural', value: 'Female' },
  { name: 'Eric (Male)', id: 'en-US-EricNeural', value: 'Male' },
  { name: 'Jacob (Male)', id: 'en-US-JacobNeural', value: 'Male' },
  {
    name: 'JennyMultilingual (Female)',
    id: 'en-US-JennyMultilingualNeural4',
    value: 'Female',
  },
  { name: 'Michelle (Female)', id: 'en-US-MichelleNeural', value: 'Female' },
  { name: 'Monica (Female)', id: 'en-US-MonicaNeural', value: 'Female' },
  { name: 'Roger (Male)', id: 'en-US-RogerNeural', value: 'Male' },
  {
    name: 'RyanMultilingual (Male)',
    id: 'en-US-RyanMultilingualNeural4',
    value: 'Male',
  },
  {
    name: 'SteffanMultilingual (Male)',
    id: 'en-US-SteffanMultilingualNeural4',
    value: 'Male',
  },
  { name: 'Steffan (Male)', id: 'en-US-SteffanNeural', value: 'Male' },
];
export const AZURE_LANGUAGE = [
  {
    code: 'en-US',
    name: 'en-US',
  },
  {
    code: 'en-IN',
    name: 'en-IN',
  },
];
// export const AZURE_ENCODINGS = [
//   {
//     name: 'Raw 24Khz 16BitMono PCM',
//     id: 'Raw24Khz16BitMonoPcm',
//     value: 'Raw24Khz16BitMonoPcm',
//   },
//   {
//     name: 'Raw 8Khz 8BitMono MULaw',
//     id: 'Raw8Khz8BitMonoMULaw',
//     value: 'Raw8Khz8BitMonoMULaw',
//   },
// ];

export const GetAzureDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];
  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'speak.language',
    'speak.voice.id',
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
    AZURE_LANGUAGE.some(l => l.code === value),
  );

  addMetadata('speak.voice.id', 'en-IN-AartiIndicNeural', value =>
    AZURE_VOICE.some(v => v.id === value),
  );

  // Only return metadata for the keys we want to keep
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('speaker.')),
  ];
};

export const ValidateAzureOptions = (options: Metadata[]): boolean => {
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
    !AZURE_LANGUAGE.some(lang => lang.code === languageOption.getValue())
  ) {
    return false;
  }

  // Validate voice
  const voiceOption = options.find(opt => opt.getKey() === 'speak.voice.id');
  if (
    !voiceOption ||
    !AZURE_VOICE.some(voice => voice.id === voiceOption.getValue())
  ) {
    return false;
  }

  return true;
};
