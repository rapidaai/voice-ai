import React, { useRef, useEffect } from 'react';
import Editor, { OnChange, OnMount } from '@monaco-editor/react';
import { useDarkMode } from '@/context/dark-mode-context';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';

export type JsonEditorProps = {
  value?: string;
  onChange?: (value: string) => void;
  onFocus?: () => void;
  onBlur?: () => void;
  editable?: boolean;
  height?: string;
  className?: string;
  placeholder?: string;
};

export const JsonEditor: React.FC<JsonEditorProps> = ({
  value = '',
  onChange,
  onFocus,
  onBlur,
  editable = true,
  className,
  placeholder = '',
  height = '200px',
}) => {
  const { isDarkMode } = useDarkMode();
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);

  const handleEditorDidMount: OnMount = (editor, monaco) => {
    editorRef.current = editor;

    if (placeholder && editor.getValue() === '') {
      new PlaceholderContentWidget(placeholder, editor, monaco);
    }

    editor.onDidFocusEditorWidget(() => onFocus?.());
    editor.onDidBlurEditorWidget(() => onBlur?.());

    if (value) {
      editor.setValue(value);
    }
  };

  useEffect(() => {
    if (editorRef.current) {
      const currentValue = editorRef.current.getValue();
      if (currentValue !== value) {
        editorRef.current.setValue(value);
        console.log('Value updated in effect:', value);
      }
    }
  }, [value]);

  const handleChange: OnChange = newValue => {
    if (onChange && newValue !== undefined) {
      onChange(newValue);
      console.log('Editor content changed:', newValue);
    }
  };

  return (
    <Editor
      height={height}
      language="json"
      className={className}
      value={value}
      onMount={handleEditorDidMount}
      onChange={handleChange}
      theme={isDarkMode ? 'vs-dark' : 'vs'}
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

class PlaceholderContentWidget implements monaco.editor.IContentWidget {
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
      this.domNode.style.pointerEvents = 'auto';
      this.domNode.style.cursor = 'text';
      this.domNode.onclick = () => {
        this.editor.focus();
      };
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

export default JsonEditor;
