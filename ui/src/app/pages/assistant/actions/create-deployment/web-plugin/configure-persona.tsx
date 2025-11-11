import { IntArrayImage } from '@/app/components/base/images/int-array-imge';
import { FormLabel } from '@/app/components/form-label';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Select } from '@/app/components/Form/Select';
import { FileUploadIcon } from '@/app/components/Icon/file-upload';
import { InputGroup } from '@/app/components/input-group';
import { cn } from '@/styles/media';
import { FC, useCallback, useState } from 'react';
import { useDropzone } from 'react-dropzone';

/**
 * Persona configure
 * an interface provide the props
 */
export interface PersonaConfig {
  name?: string;
  role?: string;
  avatarUrl?: string;
  avatar?: {
    file: Uint8Array;
    type: string;
    size: number;
    name: string;
  };
  tone?: string;
  expertise?: string;
}

/**
 *
 * @param param0
 * @returns
 */
export const ConfigurePersona: FC<{
  personaConfig: PersonaConfig;
  onChangePersona: (p: PersonaConfig) => void;
}> = ({ personaConfig, onChangePersona }) => {
  /**
   *
   * @param k
   * @param v
   */
  const handleInputChange = (k: string, v) => {
    onChangePersona({ ...personaConfig, [k]: v });
  };

  /**
   *
   */
  const onDrop = useCallback(
    acceptedFiles => {
      if (acceptedFiles.length) {
        const file = acceptedFiles[0]; // Take only the first file since multiple is false
        if (file) {
          const reader = new FileReader();
          reader.onload = () => {
            // Make sure we're not accidentally resetting anything here
            handleInputChange('avatar', {
              file: new Uint8Array(reader.result as ArrayBuffer),
              type: file.type,
              size: file.size,
              name: file.name,
            });
          };
          reader.readAsArrayBuffer(file);
        }
      }
    },
    [handleInputChange],
  ); // Add handleInputChange to the dependency array

  /**
   *
   */
  const { getRootProps, getInputProps, open } = useDropzone({
    onDrop,
    maxFiles: 1,
    accept: {
      'image/jpeg': [],
      'image/png': [],
    },
    multiple: false,
    noClick: false,
  });

  /**
   *
   */
  return (
    <InputGroup title="Appearance">
      <div className={cn('p-6 flex gap-8')}>
        <div className="flex items-center justify-center flex-col px-8 gap-1">
          <div
            {...getRootProps()}
            className="group relative w-[90px] h-[90px] p-px border dark:border-gray-700 bg-gray-100 dark:bg-gray-800 overflow-hidden"
          >
            {personaConfig.avatar ? (
              <IntArrayImage
                imageData={personaConfig.avatar.file}
                className="w-full h-full object-cover rounded-[2px]"
              />
            ) : (
              <div className="flex items-center justify-center w-full h-full">
                {personaConfig.avatarUrl ? (
                  <img
                    className="w-full h-full object-cover rounded-[2px]"
                    alt="Assistant Icon"
                    src={personaConfig.avatarUrl}
                  />
                ) : (
                  <svg
                    className="w-6 h-6 opacity-60"
                    fill="none"
                    strokeWidth={1.5}
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                    />
                  </svg>
                )}
              </div>
            )}
            <div className="absolute left-0 right-0 bottom-0 top-0 group-hover:flex hidden backdrop-blur-xs rounded-[2px] items-center justify-center text-blue-500">
              <FileUploadIcon />
            </div>
            <input
              className="hidden"
              type="file"
              name="icon"
              multiple
              {...getInputProps()}
            />
          </div>
          <button
            type="button"
            onClick={open}
            className="h-fit py-1.5 bg-white opacity-70 border hover:shadow-sm font-medium dark:bg-gray-900 text-sm w-full"
          >
            Choose
          </button>
        </div>
        <div className="grid grid-cols-1 gap-x-6 gap-y-6 md:grid-cols-2 flex-1 grow">
          <FieldSet>
            <FormLabel>Name</FormLabel>
            <Input
              maxLength={32}
              className="bg-light-background"
              placeholder="e.g. Alex"
              type="text"
              required
              value={personaConfig.name || ''}
              onChange={e => handleInputChange('name', e.target.value)}
            />
            <div className="mb-3 text-sm">
              Choose a name that reflects the agent's purpose and creates a
              relatable persona.
            </div>
          </FieldSet>
        </div>
      </div>
    </InputGroup>
  );
};
