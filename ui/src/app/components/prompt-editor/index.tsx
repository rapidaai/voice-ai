import { useEffect, useRef } from 'react';
import Editor, { OnChange, OnMount } from '@monaco-editor/react';
import { useTheme } from '@/theme/theme-provider';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import {
  extractPromptVariableQuery,
  getPromptVariableSuggestions,
} from '@/app/components/prompt-editor/suggestions';

export type PromptEditorProps = {
  value?: string;
  onChange?: (value: string) => void;
  onFocus?: () => void;
  onBlur?: () => void;
  editable?: boolean;
  height?: string;
  className?: string;
  placeholder?: string;
  enableReservedVariableSuggestions?: boolean;
};

const PromptEditor = ({
  value = '',
  onChange,
  onFocus,
  onBlur,
  editable = true,
  className,
  placeholder = '',
  enableReservedVariableSuggestions = false,
}: PromptEditorProps) => {
  const { resolvedMode } = useTheme();
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const completionProviderRef = useRef<monaco.IDisposable | null>(null);

  const handleEditorDidMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;

    completionProviderRef.current?.dispose();
    completionProviderRef.current = null;
    if (enableReservedVariableSuggestions) {
      completionProviderRef.current =
        monaco.languages.registerCompletionItemProvider('twig', {
          triggerCharacters: ['{', '.'],
          provideCompletionItems(model, position) {
            const linePrefix = model
              .getLineContent(position.lineNumber)
              .slice(0, position.column - 1);
            const query = extractPromptVariableQuery(linePrefix);

            if (query === null) {
              return { suggestions: [] };
            }

            const startColumn = position.column - query.length;
            const range = new monaco.Range(
              position.lineNumber,
              startColumn,
              position.lineNumber,
              position.column,
            );

            const suggestions = getPromptVariableSuggestions(linePrefix).map(
              (item, idx) => ({
                label: item.label,
                kind: monaco.languages.CompletionItemKind.Variable,
                insertText: item.insertText,
                detail: 'Reserved variable',
                documentation: {
                  value: item.description,
                },
                range,
                sortText: `0${idx}`,
              }),
            );

            return { suggestions };
          },
        });
    }

    if (placeholder && editor.getValue() === '') {
      new PlaceholderContentWidget(placeholder, editor, monaco);
    }

    editor.onDidFocusEditorWidget(() => onFocus?.());
    editor.onDidBlurEditorWidget(() => onBlur?.());
    editor.onDidChangeModelContent(() => {
      if (!enableReservedVariableSuggestions) return;
      const position = editor.getPosition();
      if (!position) return;

      const linePrefix = editor
        .getModel()
        ?.getLineContent(position.lineNumber)
        .slice(0, position.column - 1);
      if (!linePrefix) return;

      if (linePrefix.endsWith('{{')) {
        editor.trigger('prompt-editor', 'editor.action.triggerSuggest', {});
      }
    });

    if (value) {
      editor.setValue(value);
    }
  };

  useEffect(() => {
    if (editorRef.current) {
      const currentValue = editorRef.current.getValue();
      if (currentValue !== value) {
        editorRef.current.setValue(value);
      }
    }
  }, [value]);

  useEffect(() => {
    return () => {
      completionProviderRef.current?.dispose();
    };
  }, []);

  const handleChange: OnChange = newValue => {
    if (onChange && newValue !== undefined) {
      onChange(newValue);
    }
  };
  return (
    <Editor
      language="twig"
      className={className}
      defaultValue={value}
      onMount={handleEditorDidMount}
      onChange={handleChange}
      theme={resolvedMode === 'dark' ? 'vs-dark' : 'vs'}
      options={{
        readOnly: !editable,
        minimap: { enabled: false },
        wordWrap: 'on',
        lineNumbersMinChars: 0,
        lineNumbers: 'off',
        tabSize: 2,
        fontSize: 15,
        glyphMargin: false,
        folding: false,
        lineDecorationsWidth: 0,
        scrollbar: {
          vertical: 'hidden',
          horizontal: 'hidden',
        },
      }}
    />
  );
};

export default PromptEditor;

class PlaceholderContentWidget {
  static ID = 'editor.widget.placeholderHint';
  private domNode?: HTMLDivElement;

  constructor(
    private placeholder: string,
    private editor: monaco.editor.IStandaloneCodeEditor,
    private mEditor: typeof monaco,
  ) {
    this.editor.onDidChangeModelContent(() => this.onDidChangeModelContent());
    this.onDidChangeModelContent();
  }

  onDidChangeModelContent() {
    if (this.editor.getValue() === '') {
      this.editor.addContentWidget(this);
    } else {
      this.editor.removeContentWidget(this);
    }
  }

  getId() {
    return PlaceholderContentWidget.ID;
  }

  getDomNode() {
    if (!this.domNode) {
      this.domNode = document.createElement('div');
      this.domNode.innerText = this.placeholder;
      this.domNode.className = 'dark:text-gray-700 text-gray-400 relative!';
      this.domNode.style.pointerEvents = 'auto'; // allow click
      this.domNode.style.cursor = 'text'; // make it look like editable text
      this.domNode.onclick = () => {
        this.editor.focus();
      };
      //   this.editor.applyFontInfo(this.domNode);
    }

    return this.domNode;
  }

  getPosition(): monaco.editor.IContentWidgetPosition {
    return {
      position: { lineNumber: 1, column: 1 },
      preference: [this.mEditor.editor.ContentWidgetPositionPreference.EXACT],
    };
  }

  dispose() {
    this.editor.removeContentWidget(this);
  }
}
