import { Metadata } from '@rapidaai/react';
import { InputGroup } from '@/app/components/input-group';
import { CloudStorageProvider } from '@/app/components/providers/storage';
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
  /**
   *
   */
  return (
    <>
      {allowed.includes('text') && (
        <InputGroup title="Text Messages">
          <CloudStorageProvider
            key={'text'}
            onChangeConfig={onChangeTextConfig}
            config={textConfig}
          />
        </InputGroup>
      )}
      {allowed.includes('audio') && (
        <InputGroup title="Audio Recording">
          <CloudStorageProvider
            key={'audio'}
            onChangeConfig={onChangeAudioConfig}
            config={audioConfig}
          />
        </InputGroup>
      )}
    </>
  );
};
