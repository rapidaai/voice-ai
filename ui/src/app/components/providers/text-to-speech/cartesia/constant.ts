/**
 * Rapida – Open Source Voice AI Orchestration Platform
 * Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
 * Licensed under a modified GPL-2.0. See the LICENSE file for details.
 */
import { CARTESIA_LANGUAGE, CARTESIA_TEXT_TO_SPEECH_MODEL } from '@/providers';
import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const GetCartesiaDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];

  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'speak.language',
    'speak.voice.id',
    'speak.model',
    'speak.voice.__experimental_controls.speed',
    'speak.voice.__experimental_controls.emotion',
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
  addMetadata('speak.language');
  addMetadata('speak.voice.id');
  addMetadata('speak.model');
  addMetadata('speak.voice.__experimental_controls.speed', '', value => true);
  addMetadata('speak.voice.__experimental_controls.emotion', '', value => true);

  // Only return metadata for the keys we want to keep and non-speak metadata
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('speaker.')),
  ];
};
export const ValidateCartesiaOptions = (
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
    return 'Please select a valid credential for text to speech.';
  }

  const voiceID = options.find(opt => opt.getKey() === 'speak.voice.id');
  if (!voiceID || !voiceID.getValue() || voiceID.getValue().length === 0) {
    return 'Please select a valid voice for text to speech.';
  }

  const validations = [
    {
      key: 'speak.language',
      validator: CARTESIA_LANGUAGE(),
      field: 'code',
      errorMessage: 'Please select a valid language for text to speech.',
    },
    {
      key: 'speak.model',
      validator: CARTESIA_TEXT_TO_SPEECH_MODEL(),
      field: 'id',
      errorMessage: 'Please select a valid model for text to speech.',
    },
  ];

  for (const { key, validator, field, errorMessage } of validations) {
    const option = options.find(opt => opt.getKey() === key);
    if (!option || !validator.some(item => item[field] === option.getValue())) {
      return errorMessage;
    }
  }

  return undefined;
};
