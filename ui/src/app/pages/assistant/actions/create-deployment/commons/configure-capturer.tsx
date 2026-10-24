import { Metadata } from '@rapidaai/react';
import { CloudStorageProvider } from '@/app/components/domain/providers/storage';
import { Text } from '@/app/components/ui/primitives/text';

export interface StorageConfig {
  providerId: string;
  provider: string;
  parameters: Metadata[];
}

export const ConfigureCapturer: React.FC<{
  allowed: string[];
  onChangeAudioConfig: (config: StorageConfig | null) => void;
  audioConfig: StorageConfig | null;
  onChangeTextConfig: (config: StorageConfig | null) => void;
  textConfig: StorageConfig | null;
}> = ({
  allowed,
  onChangeAudioConfig,
  audioConfig,
  onChangeTextConfig,
  textConfig,
}) => {
  return (
    <>
      {allowed.includes('text') && (
        <div className="border-b dark:border-gray-800">
          <div className="px-5 py-4 border-b dark:border-gray-800">
            <Text as="h2" variant="heading-compact-01">
              Text Messages
            </Text>
            <Text
              as="p"
              variant="helper-text-01"
              className="mt-0.5 text-[var(--cds-text-secondary)]"
            >
              Store conversation transcripts to a cloud storage provider.
            </Text>
          </div>
          <div className="px-5 py-5">
            <CloudStorageProvider
              key={'text'}
              onChangeConfig={onChangeTextConfig}
              config={textConfig}
            />
          </div>
        </div>
      )}
      {allowed.includes('audio') && (
        <div className="border-b dark:border-gray-800">
          <div className="px-5 py-4 border-b dark:border-gray-800">
            <Text as="h2" variant="heading-compact-01">
              Audio Recording
            </Text>
            <Text
              as="p"
              variant="helper-text-01"
              className="mt-0.5 text-[var(--cds-text-secondary)]"
            >
              Save call audio recordings to a cloud storage provider.
            </Text>
          </div>
          <div className="px-5 py-5">
            <CloudStorageProvider
              key={'audio'}
              onChangeConfig={onChangeAudioConfig}
              config={audioConfig}
            />
          </div>
        </div>
      )}
    </>
  );
};
