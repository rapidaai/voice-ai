import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import PromptEditor from '../prompt-editor';

let mockResolvedMode = 'light';
let mockEditorProps: any;
let mockEditor: any;
let mockMonaco: any;
let mockDisposable: { dispose: jest.Mock };
let mockFocusHandlers: Array<() => void> = [];
let mockBlurHandlers: Array<() => void> = [];
let mockContentHandlers: Array<() => void> = [];
let mockEditorValue = '';
let mockLinePrefix = '';
let mockPosition: { lineNumber: number; column: number } | null = {
  lineNumber: 1,
  column: 1,
};
let mockMountEditor = true;

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({ resolvedMode: mockResolvedMode }),
}));

jest.mock('monaco-editor/esm/vs/editor/editor.api', () => ({}));

jest.mock('@monaco-editor/react', () => {
  const ReactMock = require('react');
  const MockMonacoEditor = (props: any) => {
    mockEditorProps = props;
    ReactMock.useEffect(() => {
      if (mockMountEditor) {
        props.onMount(mockEditor, mockMonaco);
      }
    }, []);

    return (
      <textarea
        aria-label="Prompt editor"
        className={props.className}
        data-height={props.height}
        data-language={props.language}
        data-readonly={String(props.options.readOnly)}
        data-theme={props.theme}
        data-width={props.width}
        onChange={event => props.onChange(event.target.value)}
        value={props.value}
      />
    );
  };
  return {
    __esModule: true,
    default: MockMonacoEditor,
  };
});

const createEditor = () => ({
  addContentWidget: jest.fn(),
  focus: jest.fn(),
  getModel: jest.fn(() => ({
    getLineContent: jest.fn(() => mockLinePrefix),
  })),
  getPosition: jest.fn(() => mockPosition),
  getValue: jest.fn(() => mockEditorValue),
  onDidBlurEditorWidget: jest.fn(handler => {
    mockBlurHandlers.push(handler);
  }),
  onDidChangeModelContent: jest.fn(handler => {
    mockContentHandlers.push(handler);
  }),
  onDidFocusEditorWidget: jest.fn(handler => {
    mockFocusHandlers.push(handler);
  }),
  removeContentWidget: jest.fn(),
  setValue: jest.fn(value => {
    mockEditorValue = value;
  }),
  trigger: jest.fn(),
});

const createMonaco = () => {
  mockDisposable = { dispose: jest.fn() };

  return {
    Range: jest.fn((...args: number[]) => ({ args })),
    editor: {
      ContentWidgetPositionPreference: {
        EXACT: 'exact',
      },
    },
    languages: {
      CompletionItemKind: {
        Variable: 12,
      },
      registerCompletionItemProvider: jest.fn(() => mockDisposable),
    },
  };
};

beforeEach(() => {
  mockResolvedMode = 'light';
  mockEditorProps = undefined;
  mockFocusHandlers = [];
  mockBlurHandlers = [];
  mockContentHandlers = [];
  mockEditorValue = '';
  mockLinePrefix = '';
  mockPosition = { lineNumber: 1, column: 1 };
  mockMountEditor = true;
  mockEditor = createEditor();
  mockMonaco = createMonaco();
});

describe('PromptEditor', () => {
  it('renders Monaco with prompt editor defaults and forwards changes', () => {
    const onChange = jest.fn();

    render(
      <PromptEditor
        className="prompt-shell"
        editable={false}
        height="320px"
        onChange={onChange}
        value="Hello"
      />,
    );

    const editor = screen.getByLabelText('Prompt editor');
    expect(editor).toHaveAttribute('data-language', 'twig');
    expect(editor).toHaveAttribute('data-width', '100%');
    expect(editor).toHaveAttribute('data-height', '320px');
    expect(editor).toHaveAttribute('data-theme', 'vs');
    expect(editor).toHaveAttribute('data-readonly', 'true');
    expect(editor).toHaveClass('prompt-editor');
    expect(editor).toHaveClass('prompt-shell');
    expect(mockEditorProps.options.fontFamily).toBe(
      "'IBM Plex Sans', sans-serif",
    );
    expect(mockEditorProps.options.lineDecorationsWidth).toBe(8);
    expect(mockEditorProps.options.padding).toEqual({ top: 8, bottom: 8 });
    expect(mockEditor.setValue).toHaveBeenCalledWith('Hello');

    fireEvent.change(editor, { target: { value: 'Updated' } });

    expect(onChange).toHaveBeenCalledWith('Updated');
  });

  it('uses dark Monaco theme and ignores undefined changes', () => {
    mockResolvedMode = 'dark';
    const onChange = jest.fn();

    render(<PromptEditor onChange={onChange} />);

    expect(screen.getByLabelText('Prompt editor')).toHaveAttribute(
      'data-theme',
      'vs-dark',
    );

    mockEditorProps.onChange(undefined);

    expect(onChange).not.toHaveBeenCalled();
  });

  it('registers reserved variable suggestions and disposes them on unmount', () => {
    const { unmount } = render(
      <PromptEditor enableReservedVariableSuggestions />,
    );

    expect(
      mockMonaco.languages.registerCompletionItemProvider,
    ).toHaveBeenCalledWith(
      'twig',
      expect.objectContaining({ triggerCharacters: ['{', '.'] }),
    );

    const provider =
      mockMonaco.languages.registerCompletionItemProvider.mock.calls[0][1];
    mockLinePrefix = 'Hello {{assistant.';
    mockPosition = { lineNumber: 1, column: mockLinePrefix.length + 1 };

    const completions = provider.provideCompletionItems(
      mockEditor.getModel(),
      mockPosition,
    );

    expect(completions.suggestions.length).toBeGreaterThan(0);
    expect(completions.suggestions[0]).toEqual(
      expect.objectContaining({
        detail: 'Reserved variable',
        kind: 12,
        sortText: '00',
      }),
    );
    expect(mockMonaco.Range).toHaveBeenCalledWith(
      1,
      mockPosition.column - 'assistant.'.length,
      1,
      mockPosition.column,
    );

    expect(
      provider.provideCompletionItems(
        {
          getLineContent: () => 'Hello',
        },
        { lineNumber: 1, column: 6 },
      ),
    ).toEqual({ suggestions: [] });

    unmount();

    expect(mockDisposable.dispose).toHaveBeenCalledTimes(1);
  });

  it('wires focus, blur, placeholder, and suggestion trigger behavior', () => {
    const onFocus = jest.fn();
    const onBlur = jest.fn();

    render(
      <PromptEditor
        enableReservedVariableSuggestions
        onBlur={onBlur}
        onFocus={onFocus}
        placeholder="Write a prompt"
      />,
    );

    expect(mockEditor.addContentWidget).toHaveBeenCalledTimes(1);

    const widget = mockEditor.addContentWidget.mock.calls[0][0];
    expect(widget.getId()).toBe('editor.widget.placeholderHint');
    expect(widget.getPosition()).toEqual({
      position: { lineNumber: 1, column: 1 },
      preference: ['exact'],
    });

    const domNode = widget.getDomNode();
    expect(domNode.innerText).toBe('Write a prompt');
    expect(domNode).toHaveClass('prompt-editor-placeholder');
    expect(widget.getDomNode()).toBe(domNode);

    domNode.click();
    expect(mockEditor.focus).toHaveBeenCalled();

    mockFocusHandlers.forEach(handler => handler());
    mockBlurHandlers.forEach(handler => handler());
    expect(onFocus).toHaveBeenCalledTimes(1);
    expect(onBlur).toHaveBeenCalledTimes(1);

    mockLinePrefix = 'Hello {{';
    mockPosition = { lineNumber: 1, column: mockLinePrefix.length + 1 };
    mockContentHandlers.forEach(handler => handler());
    expect(mockEditor.trigger).toHaveBeenCalledWith(
      'prompt-editor',
      'editor.action.triggerSuggest',
      {},
    );

    mockEditorValue = 'typed';
    mockContentHandlers.forEach(handler => handler());
    expect(mockEditor.removeContentWidget).toHaveBeenCalledWith(widget);
  });

  it('does not trigger suggestions without position or line prefix', () => {
    render(<PromptEditor enableReservedVariableSuggestions />);

    mockPosition = null;
    mockContentHandlers.forEach(handler => handler());

    mockPosition = { lineNumber: 1, column: 1 };
    mockLinePrefix = '';
    mockContentHandlers.forEach(handler => handler());

    expect(mockEditor.trigger).not.toHaveBeenCalled();
  });

  it('does not trigger suggestions when suggestions are disabled or prefix is incomplete', () => {
    render(<PromptEditor />);

    mockLinePrefix = 'Hello {{';
    mockPosition = { lineNumber: 1, column: mockLinePrefix.length + 1 };
    mockContentHandlers.forEach(handler => handler());

    expect(mockEditor.trigger).not.toHaveBeenCalled();

    mockContentHandlers = [];
    render(<PromptEditor enableReservedVariableSuggestions />);

    mockLinePrefix = 'Hello {{assistant';
    mockPosition = { lineNumber: 1, column: mockLinePrefix.length + 1 };
    mockContentHandlers.forEach(handler => handler());

    expect(mockEditor.trigger).not.toHaveBeenCalled();
  });

  it('does not sync values before Monaco mounts', () => {
    mockMountEditor = false;

    render(<PromptEditor value="Draft prompt" />);

    expect(mockEditor.setValue).not.toHaveBeenCalled();
  });

  it('syncs external value changes after mount', () => {
    mockEditorValue = 'Old';

    const { rerender } = render(<PromptEditor value="Old" />);
    mockEditor.setValue.mockClear();

    rerender(<PromptEditor value="New" />);

    expect(mockEditor.setValue).toHaveBeenCalledWith('New');
  });
});
