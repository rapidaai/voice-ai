import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const GetDeepgramDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];
  const keysToKeep = ['rapida.credential_id', 'speak.voice.id'];
  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  addMetadata('rapida.credential_id');
  // Set voice
  addMetadata('speak.voice.id');
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('speaker.')),
  ];
};

export const ValidateDeepgramOptions = (
  options: Metadata[],
): string | undefined => {
  const credentialID = options.find(
    opt => opt.getKey() === 'rapida.credential_id',
  );
  if (
    !credentialID ||
    !credentialID.getValue() ||
    credentialID.getValue().length === 0
  ) {
    return 'Please select valid credential for text to speech.';
  }
  const voiceID = options.find(opt => opt.getKey() === 'speak.voice.id');
  if (!voiceID || !voiceID.getValue() || voiceID.getValue().length === 0) {
    return 'Please select valid voice for text to speech.';
  }
  return undefined;
};
