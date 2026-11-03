import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';

import { DeploymentAudioPanel } from '../deployment-audio-panel';

jest.mock('@/app/components/ui/primitives', () => ({
  CopyButton: ({ children }: any) => <button>{children}</button>,
}));

jest.mock('@/app/components/domain/pills/provider-model-pill', () => ({
  ProviderPill: ({ provider }: any) => <span>{provider}</span>,
}));

const option = (key: string, value: string) => ({
  getKey: () => key,
  getValue: () => value,
});

const audio = (provider: string, options: any[]) => ({
  getAudioprovider: () => provider,
  getAudiooptionsList: () => options,
});

describe('DeploymentAudioPanel', () => {
  it('renders configured speech providers and their scoped options', () => {
    render(
      <DeploymentAudioPanel
        inputAudio={
          audio('deepgram', [
            option('listen.model', 'nova-3'),
            option('speak.voice', 'ignored'),
          ]) as any
        }
        outputAudio={
          audio('cartesia', [
            option('speak.voice', 'sonic'),
            option('listen.model', 'ignored'),
          ]) as any
        }
      />,
    );

    expect(screen.getByText('deepgram')).toBeInTheDocument();
    expect(screen.getByText('cartesia')).toBeInTheDocument();
    expect(screen.getByText('Voice configuration')).toBeInTheDocument();
    expect(screen.getByText('Input provider')).toBeInTheDocument();
    expect(screen.getByText('Output provider')).toBeInTheDocument();
    expect(screen.getByText('Input model')).toBeInTheDocument();
    expect(screen.getByText('Output voice')).toBeInTheDocument();
    expect(screen.getAllByText('nova-3')).toHaveLength(2);
    expect(screen.getAllByText('sonic')).toHaveLength(2);
    expect(screen.queryByText('ignored')).not.toBeInTheDocument();
  });

  it('distinguishes a configured provider with no options from a disabled provider', () => {
    render(
      <DeploymentAudioPanel
        inputAudio={audio('deepgram', []) as any}
        outputAudio={undefined}
      />,
    );

    expect(screen.getByText('deepgram')).toBeInTheDocument();
    expect(screen.getByText('Input options')).toBeInTheDocument();
    expect(screen.getByText('Output provider')).toBeInTheDocument();
    expect(screen.getAllByText('Not configured')).toHaveLength(1);
    expect(screen.getByText('Disabled')).toBeInTheDocument();
  });
});
