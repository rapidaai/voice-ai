import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { CodeEditor } from '../code-editor';
import { CodeHighlighting } from '../code-highlighting';
import { JsonEditor } from '../json-editor';
import { MarkdownViewer } from '../markdown-viewer';

const mockEditorDisposables: Array<() => void> = [];
const mockUpdateOptions = jest.fn();
const mockAddContentWidget = jest.fn();

jest.mock('@/theme/theme-provider', () => ({
  useTheme: () => ({ resolvedMode: 'dark' }),
}));

jest.mock('@carbon/react', () => ({
  Button: ({
    className: _className,
    hasIconOnly: _hasIconOnly,
    iconDescription,
    kind: _kind,
    renderIcon: Icon,
    size: _size,
    onClick,
    ...props
  }: any) => (
    <button
      aria-label={iconDescription}
      type="button"
      onClick={onClick}
      {...props}
    >
      {Icon ? <Icon /> : null}
    </button>
  ),
}));

jest.mock('@carbon/icons-react', () => ({
  Checkmark: () => <svg data-testid="checkmark-icon" />,
  Copy: () => <svg data-testid="copy-icon" />,
  Maximize: () => <svg data-testid="maximize-icon" />,
  Minimize: () => <svg data-testid="minimize-icon" />,
}));

jest.mock('@/app/components/ui/primitives/buttons/copy-button', () => ({
  CopyButton: ({ children }: any) => (
    <button type="button" aria-label="Copy code">
      {children}
    </button>
  ),
}));

jest.mock('@uiw/react-markdown-preview', () => ({
  __esModule: true,
  default: ({ source }: any) => (
    <article data-testid="markdown">{source}</article>
  ),
}));

jest.mock('monaco-editor/esm/vs/editor/editor.api', () => ({
  editor: {
    ContentWidgetPositionPreference: {
      EXACT: 0,
    },
  },
  Range: class Range {
    constructor(
      startLineNumber: number,
      startColumn: number,
      endLineNumber: number,
      endColumn: number,
    ) {
      Object.assign(this, {
        startLineNumber,
        startColumn,
        endLineNumber,
        endColumn,
      });
    }
  },
}));

jest.mock('@monaco-editor/react', () => {
  const React = require('react');
  const MockMonacoEditor = ({
    className,
    language,
    onChange,
    onMount,
    options,
    theme,
    value = '',
  }: any) => {
    React.useEffect(() => {
      let currentValue = value;
      const editor = {
        addContentWidget: mockAddContentWidget,
        focus: jest.fn(),
        getModel: () => ({ uri: { toString: () => 'test://model.json' } }),
        getValue: () => currentValue,
        onDidBlurEditorWidget: jest.fn(),
        onDidChangeModelContent: (handler: () => void) => {
          mockEditorDisposables.push(handler);
          return { dispose: jest.fn() };
        },
        onDidFocusEditorWidget: jest.fn(),
        removeContentWidget: jest.fn(),
        setValue: (next: string) => {
          currentValue = next;
        },
        updateOptions: mockUpdateOptions,
      };
      onMount?.(editor, {
        editor: {
          ContentWidgetPositionPreference: {
            EXACT: 0,
          },
        },
      });
    }, []);

    return (
      <textarea
        aria-label={`${language}-editor`}
        className={className}
        data-font-family={options?.fontFamily}
        data-line-padding={options?.lineDecorationsWidth}
        data-padding-bottom={options?.padding?.bottom}
        data-padding-top={options?.padding?.top}
        data-readonly={String(options?.readOnly ?? false)}
        data-theme={theme}
        value={value}
        onChange={event => onChange?.(event.target.value)}
      />
    );
  };
  return {
    __esModule: true,
    default: MockMonacoEditor,
  };
});

describe('editor components', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    Object.assign(navigator, {
      clipboard: {
        writeText: jest.fn(),
      },
    });
    mockUpdateOptions.mockClear();
    mockAddContentWidget.mockClear();
    mockEditorDisposables.length = 0;
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('renders JsonEditor with theme and forwards changes', () => {
    const onChange = jest.fn();
    const dispose = jest.fn();

    const { unmount } = render(
      <JsonEditor
        className="consumer-class"
        value=""
        placeholder="Enter JSON"
        onChange={onChange}
        configureEditor={() => ({ dispose })}
      />,
    );

    const editor = screen.getByLabelText('json-editor');
    expect(editor).toHaveAttribute('data-theme', 'vs-dark');
    expect(editor).toHaveAttribute('data-font-family', 'IBM Plex Mono');
    expect(editor).toHaveAttribute('data-line-padding', '8');
    expect(editor).toHaveAttribute('data-padding-bottom', '8');
    expect(editor).toHaveAttribute('data-padding-top', '8');
    expect(editor).toHaveClass('json-editor', 'consumer-class');
    expect(editor.closest('.json-editor-surface')).toBeInTheDocument();
    const placeholderWidget = mockAddContentWidget.mock.calls[0][0];
    expect(placeholderWidget.getDomNode()).toHaveClass(
      'json-editor-placeholder',
    );

    fireEvent.change(editor, { target: { value: '{"a":2}' } });
    expect(onChange).toHaveBeenCalledWith('{"a":2}');

    unmount();
    expect(dispose).toHaveBeenCalledTimes(1);
  });

  it('renders CodeEditor controls and handles copy, expand, and edits', () => {
    const onChange = jest.fn();

    render(
      <CodeEditor
        labelText="Parameters"
        helperText="JSON body"
        placeholder="Enter JSON"
        value='{"enabled":true}'
        onChange={onChange}
      />,
    );

    expect(screen.getByText('Parameters')).toBeInTheDocument();
    expect(screen.getByText('JSON body')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('json-editor'), {
      target: { value: '{"enabled":false}' },
    });
    expect(onChange).toHaveBeenCalledWith('{"enabled":false}');

    fireEvent.click(screen.getByRole('button', { name: 'Copy' }));
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith(
      '{"enabled":true}',
    );
    expect(screen.getByTestId('checkmark-icon')).toBeInTheDocument();

    act(() => {
      jest.advanceTimersByTime(4000);
    });
    expect(screen.getByTestId('copy-icon')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Maximize' }));
    expect(
      screen.getByRole('button', { name: 'Minimize' }),
    ).toBeInTheDocument();
  });

  it('renders read-only highlighted code with copy action', () => {
    render(<CodeHighlighting language="html" code="<main>Hello</main>" />);

    expect(screen.getByLabelText('html-editor')).toHaveValue(
      '<main>Hello</main>',
    );
    expect(screen.getByLabelText('html-editor')).toHaveAttribute(
      'data-readonly',
      'true',
    );
    expect(screen.getByLabelText('html-editor')).toHaveAttribute(
      'data-font-family',
      'IBM Plex Mono',
    );
    expect(screen.getByLabelText('html-editor')).toHaveAttribute(
      'data-line-padding',
      '8',
    );
    expect(screen.getByLabelText('html-editor')).toHaveAttribute(
      'data-padding-bottom',
      '8',
    );
    expect(screen.getByLabelText('html-editor')).toHaveAttribute(
      'data-padding-top',
      '8',
    );
    expect(screen.getByLabelText('html-editor')).toHaveClass(
      'code-highlighting-editor',
    );
    expect(screen.getByRole('button', { name: 'Copy code' })).toHaveTextContent(
      '<main>Hello</main>',
    );
  });

  it('renders markdown content with paragraph spacing', () => {
    render(<MarkdownViewer text={'Line one\nLine two'} />);

    expect(screen.getByTestId('markdown').textContent).toBe(
      'Line one\n\nLine two',
    );
  });
});
