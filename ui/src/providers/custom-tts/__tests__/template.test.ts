import {
  CUSTOM_TTS_DEFAULT_REQUEST_RULES_EXAMPLE,
  CUSTOM_TTS_DSL_VARIABLES,
  CUSTOM_TTS_QUERY_PARAMS_EXAMPLE,
  CUSTOM_TTS_REQUEST_RULES_DONE_EXAMPLE,
  CUSTOM_TTS_REQUEST_RULES_INTERRUPT_EXAMPLE,
  CUSTOM_TTS_REQUEST_RULES_KEY,
  CUSTOM_TTS_RESPONSE_RULES_BINARY_EXAMPLE,
  CUSTOM_TTS_RESPONSE_RULES_JSON_AUDIO_EXAMPLE,
  getCustomTtsFlowMode,
  parseCustomTtsRequestRules,
  parseCustomTtsResponseFrame,
  parseCustomTtsResponseRules,
  renderCustomTtsQueryParams,
  renderCustomTtsRequestRuleBody,
  resolveCustomTtsFlowMode,
  validateCustomTtsQueryParams,
  validateCustomTtsRequestRules,
  validateCustomTtsResponseRules,
} from '../contract';

const metadata = (key: string, value: string) => ({
  getKey: () => key,
  getValue: () => value,
});

describe('custom-tts websocket DSL helpers', () => {
  it('renders query params and request rule bodies from the shared DSL', () => {
    const [textRule] = parseCustomTtsRequestRules(
      CUSTOM_TTS_DEFAULT_REQUEST_RULES_EXAMPLE,
    );
    const renderedRequestBody = renderCustomTtsRequestRuleBody(textRule, {
      config: {
        voice: { id: 'voice-9' },
        model: 'sonic-2',
        language: 'en-US',
        audio: {
          encoding: 'LINEAR16',
          sample_rate: '16000',
        },
      },
      packet: {
        kind: 'text',
        message_id: 'msg-7',
        text: 'Hello "world"',
      },
    });

    expect(renderedRequestBody).toEqual({
      text: 'Hello "world"',
      voice_id: 'narrator-1',
      message_id: 'msg-7',
      model: 'sonic-2',
      language: 'en-US',
      audio: {
        encoding: 'LINEAR16',
        sample_rate: 16000,
      },
    });

    expect(
      renderCustomTtsQueryParams(CUSTOM_TTS_QUERY_PARAMS_EXAMPLE, {
        text: 'Hello "world"',
        message_id: 'msg-7',
        voice_id: 'voice-9',
        model: 'sonic-2',
        language: 'en-US',
        encoding: 'LINEAR16',
        sample_rate: '16000',
      }),
    ).toEqual({
      language: 'en-US',
      model: 'sonic-2',
      voice: 'voice-9',
      message_id: 'msg-7',
      sample_rate: 16000,
    });
  });

  it('selects one-shot vs two-step mode from request rules', () => {
    expect(resolveCustomTtsFlowMode('')).toBe('one-shot');
    expect(resolveCustomTtsFlowMode('  ')).toBe('one-shot');
    expect(
      resolveCustomTtsFlowMode(CUSTOM_TTS_DEFAULT_REQUEST_RULES_EXAMPLE),
    ).toBe('one-shot');
    expect(
      resolveCustomTtsFlowMode(CUSTOM_TTS_REQUEST_RULES_DONE_EXAMPLE),
    ).toBe('two-step');

    expect(
      getCustomTtsFlowMode([
        metadata(
          CUSTOM_TTS_REQUEST_RULES_KEY,
          CUSTOM_TTS_REQUEST_RULES_DONE_EXAMPLE,
        ),
      ]),
    ).toBe('two-step');
  });

  it('rejects unsupported DSL variables, bad query param values, and request rules without text', () => {
    const queryTextResult = validateCustomTtsQueryParams(
      '{"text":{"$var":"text"}}',
      'Query Parameters',
    );
    expect(queryTextResult).toContain('Unsupported custom tts variable "text"');

    const variableResult = validateCustomTtsQueryParams(
      '{"message_id":{"$var":"request_id"}}',
      'Query Parameters',
    );

    expect(variableResult).toContain('"request_id"');
    for (const variable of CUSTOM_TTS_DSL_VARIABLES) {
      expect(variableResult).toContain(variable);
    }

    expect(
      validateCustomTtsQueryParams(
        '{"audio":{"encoding":{"$var":"encoding"}}}',
        'Query Parameters',
      ),
    ).toBe(
      'Custom TTS query parameters values must resolve to strings, numbers, or booleans.',
    );

    expect(
      validateCustomTtsRequestRules(
        '[{"when":{"packet":"done"},"send":{"frame":"json","body":{"type":"done"}}}]',
      ),
    ).toBe(
      'Custom TTS request rules must contain at least one rule with when.packet "text".',
    );
  });

  it('parses text, done, and interrupt request rule snippets', () => {
    const doneRules = parseCustomTtsRequestRules(
      CUSTOM_TTS_REQUEST_RULES_DONE_EXAMPLE,
    );
    expect(doneRules.map(rule => rule.when.packet)).toEqual(['text', 'done']);

    const interruptRules = parseCustomTtsRequestRules(
      CUSTOM_TTS_REQUEST_RULES_INTERRUPT_EXAMPLE,
    );
    expect(interruptRules.map(rule => rule.when.packet)).toEqual([
      'text',
      'interrupt',
    ]);
  });

  it('validates and parses the binary-audio response rules contract', () => {
    expect(
      validateCustomTtsResponseRules(CUSTOM_TTS_RESPONSE_RULES_BINARY_EXAMPLE),
    ).toBeUndefined();

    const parser = parseCustomTtsResponseRules(
      CUSTOM_TTS_RESPONSE_RULES_BINARY_EXAMPLE,
    );

    expect(
      parseCustomTtsResponseFrame(new Uint8Array([1, 2, 3]), parser),
    ).toEqual({
      kind: 'audio',
      audio: new Uint8Array([1, 2, 3]),
      payload: new Uint8Array([1, 2, 3]),
    });

    expect(
      parseCustomTtsResponseFrame(
        '{"type":"done","message_id":"msg-7"}',
        parser,
      ),
    ).toEqual({
      kind: 'done',
      messageId: 'msg-7',
      payload: {
        type: 'done',
        message_id: 'msg-7',
      },
      done: true,
    });
  });

  it('parses JSON audio frames and error frames from ordered response rules', () => {
    const parser = parseCustomTtsResponseRules(
      CUSTOM_TTS_RESPONSE_RULES_JSON_AUDIO_EXAMPLE,
    );

    expect(
      parseCustomTtsResponseFrame(
        '{"type":"chunk","message_id":"msg-8","audio":"AAEC"}',
        parser,
      ),
    ).toEqual({
      kind: 'audio',
      messageId: 'msg-8',
      payload: {
        type: 'chunk',
        message_id: 'msg-8',
        audio: 'AAEC',
      },
      audio: new Uint8Array([0, 1, 2]),
    });

    expect(
      parseCustomTtsResponseFrame(
        '{"type":"error","message_id":"msg-9","error":{"message":"bad frame"}}',
        parser,
      ),
    ).toEqual({
      kind: 'error',
      messageId: 'msg-9',
      payload: {
        type: 'error',
        message_id: 'msg-9',
        error: {
          message: 'bad frame',
        },
      },
      error: 'bad frame',
      done: true,
    });
  });
});
