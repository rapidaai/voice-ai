/**
 * Rapida – Open Source Voice AI Orchestration Platform
 * Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
 * Licensed under a modified GPL-2.0. See the LICENSE file for details.
 */
import { ELEVENLABS_LANGUAGE, ELEVENLABS_MODEL } from '@/providers';
import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const GetElevanLabDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];

  // Define the keys we want to keep
  const keysToKeep = [
    'rapida.credential_id',
    'speak.language',
    'speak.voice.id',
    'speak.model',
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
  addMetadata('speak.language');

  // Set voice
  addMetadata('speak.voice.id');

  // Set model
  addMetadata('speak.model');

  // Only return metadata for the keys we want to keep and non-speak metadata
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('speaker.')),
  ];
};

export const ValidateElevanLabOptions = (
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
    return 'Please select a valid voice ID for text to speech.';
  }

  const validations = [
    {
      key: 'speak.language',
      validator: ELEVENLABS_LANGUAGE(),
      field: 'language_id',
      errorMessage: 'Please select valid language for text to speech.',
    },
    {
      key: 'speak.model',
      validator: ELEVENLABS_MODEL(),
      field: 'model_id',
      errorMessage: 'Please select valid model for text to speech.',
    },
  ];

  for (const { key, validator, field, errorMessage } of validations) {
    const option = options.find(opt => opt.getKey() === key);
    if (!option || !validator.some(item => item[field] === option.getValue())) {
      return errorMessage;
    }
  }

  return undefined; // Return undefined if all validations pass
};
