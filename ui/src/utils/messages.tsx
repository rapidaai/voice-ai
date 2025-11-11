import React from 'react';

/**
 * This function has two roles:
 * 1) If the `id` is empty it assings something so does i18next doesn't throw error. Typescript should prevent this anyway
 * 2) It has a hand-picked name `_t` (to be short) and should only be used while using objects instead of strings for translation keys
 * `internals/extractMessages/stringfyTranslations.js` script converts this to `t('a.b.c')` style before `i18next-scanner` scans the file contents
 * so that our json objects can also be recognized by the scanner.
 */
export const _t = (id: string, ...rest: any[]): [string, ...any[]] => {
  if (!id) {
    id = '_NOT_TRANSLATED_';
  }
  return [id, ...rest];
};

export const create_knowledge_success_message = (
  name: string,
): React.ReactElement => {
  return (
    <p>
      Your knowledge base <strong className="font-medium">{name}</strong> has
      been successfully created.
    </p>
  );
};

export const create_endpoint_version_success_message = (
  endpointName: string,
): React.ReactElement => {
  return (
    <p>
      New version of endpoint{' '}
      <strong className="font-medium">{endpointName}</strong> has been
      successfully created.
    </p>
  );
};

export const create_endpoint_success_message = (
  endpointName: string,
): React.ReactElement => {
  return (
    <p>
      Your endpoint <strong className="font-medium">{endpointName}</strong> has
      been successfully created.
    </p>
  );
};

export const create_assistant_version_success_message = (
  assistantName: string,
): React.ReactElement => {
  return (
    <p>
      New version of assistant{' '}
      <strong className="font-medium">{assistantName}</strong> has been
      successfully created.
    </p>
  );
};

export const create_assistant_success_message = (
  assistantName: string,
): React.ReactElement => {
  return (
    <p>
      Your assistant <strong className="font-medium">{assistantName}</strong>{' '}
      has been successfully created.
    </p>
  );
};
