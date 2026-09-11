import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { Metadata } from '@rapidaai/react';
import { EndOfSpeechProvider } from '..';
import { EOS_MODEL_PATH_KEYS, GetDefaultEOSConfig } from '../provider';
import { loadProviderConfig } from '@/providers/config-loader';

jest.mock('@/utils', () => ({
  cn: (...inputs: any[]) => inputs.filter(Boolean).join(' '),
}));

jest.mock('@/app/components/json-editor', () => ({
  JsonEditor: () => null,
}));

jest.mock('@/app/components/providers/websocket-dsl-editor', () => ({
  WebsocketDslEditor: () => null,
}));

const meta = (key: string, value: string): Metadata => {
  const metadata = new Metadata();
  metadata.setKey(key);
  metadata.setValue(value);
  return metadata;
};

describe('EOS provider runtime parity', () => {
  beforeAll(() => {
    Element.prototype.scrollIntoView = jest.fn();
  });

  afterAll(() => {
    delete (Element.prototype as Partial<Element>).scrollIntoView;
  });

  it.each(['pipecat_smart_turn_eos', 'livekit_eos'])(
    '%s renders each configured control with its hydrated default',
    provider => {
      const parameters = GetDefaultEOSConfig(provider, []);
      const onChangeParameter = jest.fn();
      render(
        <EndOfSpeechProvider
          provider={provider}
          parameters={parameters}
          onChangeProvider={jest.fn()}
          onChangeParameter={onChangeParameter}
        />,
      );

      for (const parameter of loadProviderConfig(provider)!.eos!.parameters) {
        if (parameter.type === 'slider') {
          expect(screen.getByText(parameter.label)).toBeVisible();
          expect(
            screen
              .getAllByRole('slider')
              .find(slider => slider.id === `slider-${parameter.key}`),
          ).toHaveAttribute('aria-valuenow', String(parameter.default));
        } else if (parameter.type === 'number') {
          expect(screen.getByLabelText(parameter.label)).toHaveValue(
            Number(parameter.default),
          );
        } else {
          expect(screen.getByText('English (66MB, optimized)')).toBeVisible();
        }
      }
      expect(screen.queryByText('Fallback Timeout')).not.toBeInTheDocument();
      expect(screen.queryByText('Quick Timeout')).not.toBeInTheDocument();
      expect(screen.getAllByRole('slider')).toHaveLength(3);
      expect(onChangeParameter).not.toHaveBeenCalled();
    },
  );

  it.each([
    [
      'pipecat_smart_turn_eos',
      'fallback_timeout',
      'Transcript Safety Wait (ms)',
      '900',
    ],
    ['pipecat_smart_turn_eos', 'threshold', 'Turn Completion Threshold', '0.6'],
    [
      'pipecat_smart_turn_eos',
      'extended_timeout',
      'Received Silence Limit (ms)',
      '2400',
    ],
    ['livekit_eos', 'quick_timeout', 'Minimum Endpoint Delay (ms)', '350'],
    ['livekit_eos', 'threshold', 'Turn Completion Threshold', '0.04'],
    ['livekit_eos', 'extended_timeout', 'Maximum Endpoint Delay (ms)', '2400'],
  ])(
    '%s edits %s without mutating input metadata',
    (provider, key, label, value) => {
      const parameters = GetDefaultEOSConfig(provider, [
        meta('listen.model', 'nova-3'),
        meta('microphone.eos.extended_timeout', '2000'),
      ]);
      const original = parameters.map(parameter => parameter.toObject());
      const onChangeParameter = jest.fn();
      const { container } = render(
        <EndOfSpeechProvider
          provider={provider}
          parameters={parameters}
          onChangeProvider={jest.fn()}
          onChangeParameter={onChangeParameter}
        />,
      );

      expect(
        screen
          .getAllByRole('slider')
          .find(
            slider => slider.id === 'slider-microphone.eos.extended_timeout',
          ),
      ).toHaveAttribute('aria-valuenow', '2000');
      expect(screen.getByText(label)).toBeVisible();
      const input = container.querySelector(
        `input[id="slider-microphone.eos.${key}-input-for-slider"]`,
      );
      expect(input).not.toBeNull();
      fireEvent.change(input!, { target: { value } });

      const updated = onChangeParameter.mock.calls.at(-1)![0] as Metadata[];
      expect(
        updated
          .find(parameter => parameter.getKey() === `microphone.eos.${key}`)
          ?.getValue(),
      ).toBe(value);
      expect(
        updated
          .find(parameter => parameter.getKey() === 'listen.model')
          ?.getValue(),
      ).toBe('nova-3');
      expect(parameters.map(parameter => parameter.toObject())).toEqual(
        original,
      );
    },
  );

  it('edits LiveKit history without a maximum and preserves saved model selection', () => {
    const parameters = GetDefaultEOSConfig('livekit_eos', [
      meta('microphone.eos.model', 'multilingual'),
    ]);
    const onChangeParameter = jest.fn();
    render(
      <EndOfSpeechProvider
        provider="livekit_eos"
        parameters={parameters}
        onChangeProvider={jest.fn()}
        onChangeParameter={onChangeParameter}
      />,
    );

    expect(
      screen.getByText('Multilingual (378MB, 14 languages)'),
    ).toBeVisible();
    const history = screen.getByLabelText('Maximum History Turns');
    expect(history).toHaveAttribute('type', 'number');
    expect(history).toHaveAttribute('min', '1');
    expect(history).toHaveAttribute('step', '1');
    expect(history).not.toHaveAttribute('max');
    fireEvent.change(history, { target: { value: '1000' } });

    const updated = onChangeParameter.mock.calls.at(-1)![0] as Metadata[];
    expect(
      updated
        .find(
          parameter =>
            parameter.getKey() === 'microphone.eos.max_history_turns',
        )
        ?.getValue(),
    ).toBe('1000');
    expect(
      updated
        .find(parameter => parameter.getKey() === 'microphone.eos.model')
        ?.getValue(),
    ).toBe('multilingual');
    expect(
      parameters
        .find(
          parameter =>
            parameter.getKey() === 'microphone.eos.max_history_turns',
        )
        ?.getValue(),
    ).toBe(6);
  });

  it('switches providers through the dropdown and the existing caller-owned scope cleanup', () => {
    const unrelated = meta('listen.model', 'nova-3');
    const liveKitModelPath = meta(
      'microphone.eos.livekit.model_path',
      '/models/livekit.onnx',
    );
    const liveKitTokenizerPath = meta(
      'microphone.eos.livekit.tokenizer_path',
      '/models/tokenizer.json',
    );
    const pipecatModelPath = meta(
      'microphone.eos.pipecat.model_path',
      '/models/pipecat',
    );
    let provider = 'livekit_eos';
    let parameters = GetDefaultEOSConfig(provider, [
      unrelated,
      liveKitModelPath,
      liveKitTokenizerPath,
      pipecatModelPath,
    ]);
    const onChangeProvider = jest.fn((selected: string) => {
      provider = selected;
      parameters = GetDefaultEOSConfig(
        selected,
        parameters.filter(
          parameter =>
            !parameter.getKey().startsWith('microphone.eos.') ||
            EOS_MODEL_PATH_KEYS.has(parameter.getKey()),
        ),
      );
    });
    const onChangeParameter = jest.fn();
    const { rerender } = render(
      <EndOfSpeechProvider
        provider={provider}
        parameters={parameters}
        onChangeProvider={onChangeProvider}
        onChangeParameter={onChangeParameter}
      />,
    );

    for (const [name, selected] of [
      ['Pipecat Smart Turn', 'pipecat_smart_turn_eos'],
      ['LiveKit', 'livekit_eos'],
    ]) {
      fireEvent.click(
        screen.getByRole('combobox', { name: 'End-of-speech provider' }),
      );
      fireEvent.click(
        screen.getByRole('option', { name: new RegExp(name, 'i') }),
      );
      expect(onChangeProvider).toHaveBeenLastCalledWith(selected);
      rerender(
        <EndOfSpeechProvider
          provider={provider}
          parameters={parameters}
          onChangeProvider={onChangeProvider}
          onChangeParameter={onChangeParameter}
        />,
      );
      expect(parameters).toContain(unrelated);
      expect(parameters).toContain(liveKitModelPath);
      expect(parameters).toContain(liveKitTokenizerPath);
      expect(parameters).toContain(pipecatModelPath);
      expect(
        parameters
          .find(parameter => parameter.getKey() === 'microphone.eos.provider')
          ?.getValue(),
      ).toBe(selected);
      expect(
        String(
          parameters
            .find(
              parameter =>
                parameter.getKey() === 'microphone.eos.extended_timeout',
            )
            ?.getValue(),
        ),
      ).toBe(selected === 'pipecat_smart_turn_eos' ? '4000' : '3000');
      expect(
        parameters.some(
          parameter => parameter.getKey() === 'microphone.eos.quick_timeout',
        ),
      ).toBe(selected === 'livekit_eos');
      expect(
        parameters.some(
          parameter => parameter.getKey() === 'microphone.eos.fallback_timeout',
        ),
      ).toBe(selected === 'pipecat_smart_turn_eos');
      if (selected === 'livekit_eos') {
        expect(screen.getByLabelText('Maximum History Turns')).toHaveValue(6);
      } else {
        expect(
          screen.queryByLabelText('Maximum History Turns'),
        ).not.toBeInTheDocument();
      }
      for (const parameter of loadProviderConfig(selected)!.eos!.parameters) {
        if (parameter.type !== 'slider') continue;
        expect(
          screen
            .getAllByRole('slider')
            .find(slider => slider.id === `slider-${parameter.key}`),
        ).toHaveAttribute('aria-valuenow', String(parameter.default));
      }
    }
  });
});
