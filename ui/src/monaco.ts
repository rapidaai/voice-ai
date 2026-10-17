import { loader } from '@monaco-editor/react';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker';
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker';
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker';
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker';
import typeScriptWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker';

const monacoEnvironment = {
  getWorker(_moduleId: string, label: string): Worker {
    if (label === 'json') return new jsonWorker();
    if (['css', 'scss', 'less'].includes(label)) return new cssWorker();
    if (['html', 'handlebars', 'razor'].includes(label)) return new htmlWorker();
    if (['typescript', 'javascript'].includes(label)) {
      return new typeScriptWorker();
    }
    return new editorWorker();
  },
};

(globalThis as typeof globalThis & { MonacoEnvironment: typeof monacoEnvironment })
  .MonacoEnvironment = monacoEnvironment;
loader.config({ monaco });
