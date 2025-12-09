/**
 * Rapida – Open Source Voice AI Orchestration Platform
 * Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
 * Licensed under a modified GPL-2.0. See the LICENSE file for details.
 */

import { SetMetadata } from '@/utils/metadata';
import { Metadata } from '@rapidaai/react';

export const GetGoogleDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];
  // Define the keys we want to keep
  const keysToKeep = ['rapida.credential_id', 'speak.voice.id'];

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
  addMetadata('speak.voice.id');
  // Only return metadata for the keys we want to keep
  return [
    ...mtds.filter(m => keysToKeep.includes(m.getKey())),
    ...current.filter(m => m.getKey().startsWith('speaker.')),
  ];
};

export const ValidateGoogleOptions = (
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
    return 'Please select valid google credential for text to speech.';
  }
  const voiceID = options.find(opt => opt.getKey() === 'speak.voice.id');
  if (!voiceID || !voiceID.getValue() || voiceID.getValue().length === 0) {
    return 'Please select valid voice for text to speech.';
  }

  return undefined;
};
