import { JsonEditor } from '@/app/components/ui/editor/json-editor';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import {
  getWebsocketDslEditorSuggestions,
  shouldAutoTriggerWebsocketDslSuggestions,
  WebsocketDslEditorMode,
  WebsocketDslEditorProvider,
} from './websocket-dsl-editor/suggestions';

type WebsocketDslEditorProps = {
  provider: WebsocketDslEditorProvider;
  mode: WebsocketDslEditorMode;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  ariaLabel?: string;
  className?: string;
  height?: string;
};

function getCompletionKind(
  monacoInstance: typeof monaco,
  kind: 'snippet' | 'variable' | 'value',
) {
  return kind === 'snippet'
    ? monacoInstance.languages.CompletionItemKind.Snippet
    : kind === 'variable'
      ? monacoInstance.languages.CompletionItemKind.Variable
      : monacoInstance.languages.CompletionItemKind.Value;
}

export function WebsocketDslEditor({
  provider,
  mode,
  value,
  onChange,
  placeholder,
  ariaLabel,
  className,
  height = '160px',
}: WebsocketDslEditorProps) {
  return (
    <JsonEditor
      value={value}
      onChange={onChange}
      placeholder={placeholder}
      ariaLabel={ariaLabel}
      className={className}
      height={height}
      configureEditor={(editor, monacoInstance) => {
        const modelUri = editor.getModel()?.uri.toString();
        const triggerCharacters =
          mode === 'response_rules' || mode === 'request_rules' ? ['['] : ['{'];
        const completionProvider =
          monacoInstance.languages.registerCompletionItemProvider('json', {
            triggerCharacters,
            provideCompletionItems(model, position) {
              if (model.uri.toString() !== modelUri) {
                return { suggestions: [] };
              }

              const linePrefix = model
                .getLineContent(position.lineNumber)
                .slice(0, position.column - 1);
              const suggestions = getWebsocketDslEditorSuggestions(
                provider,
                mode,
                linePrefix,
              );

              return {
                suggestions: suggestions.map((item, index) => {
                  const range =
                    item.kind !== 'snippet' && item.query !== undefined
                      ? new monacoInstance.Range(
                          position.lineNumber,
                          position.column - item.query.length,
                          position.lineNumber,
                          position.column,
                        )
                      : item.kind === 'snippet' &&
                          (linePrefix.trim().endsWith('{') ||
                            linePrefix.trim().endsWith('['))
                        ? new monacoInstance.Range(
                            position.lineNumber,
                            position.column - 1,
                            position.lineNumber,
                            position.column,
                          )
                        : new monacoInstance.Range(
                            position.lineNumber,
                            position.column,
                            position.lineNumber,
                            position.column,
                          );

                  return {
                    label: item.label,
                    kind: getCompletionKind(monacoInstance, item.kind),
                    insertText: item.insertText,
                    detail: item.detail,
                    documentation: {
                      value: item.description,
                    },
                    insertTextRules:
                      item.kind === 'snippet'
                        ? monacoInstance.languages.CompletionItemInsertTextRule
                            .InsertAsSnippet
                        : undefined,
                    range,
                    sortText: `0${index}`,
                  };
                }),
              };
            },
          });

        const contentListener = editor.onDidChangeModelContent(() => {
          const position = editor.getPosition();
          if (!position) return;

          const linePrefix = editor
            .getModel()
            ?.getLineContent(position.lineNumber)
            .slice(0, position.column - 1);
          if (!linePrefix) return;

          if (shouldAutoTriggerWebsocketDslSuggestions(mode, linePrefix)) {
            editor.trigger(
              'websocket-dsl-editor',
              'editor.action.triggerSuggest',
              {},
            );
          }
        });

        return {
          dispose() {
            contentListener.dispose();
            completionProvider.dispose();
          },
        };
      }}
    />
  );
}
