import { loader } from '@monaco-editor/react';
import * as monaco from 'monaco-editor/esm/vs/editor/editor.api';
import 'monaco-editor/esm/vs/editor/contrib/suggest/browser/suggestController';
import 'monaco-editor/esm/vs/basic-languages/html/html.contribution';
import 'monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution';
import 'monaco-editor/esm/vs/basic-languages/twig/twig.contribution';
import 'monaco-editor/esm/vs/basic-languages/typescript/typescript.contribution';
import 'monaco-editor/esm/vs/language/html/monaco.contribution';
import 'monaco-editor/esm/vs/language/json/monaco.contribution';
import 'monaco-editor/esm/vs/language/typescript/monaco.contribution';
import {
  CssWorker,
  EditorWorker,
  HtmlWorker,
  JsonWorker,
  TypeScriptWorker,
} from './monaco-workers';

const monacoEnvironment = {
  getWorker(_moduleId: string, label: string): Worker {
    if (label === 'json') return new JsonWorker();
    if (['css', 'scss', 'less'].includes(label)) return new CssWorker();
    if (['html', 'handlebars', 'razor'].includes(label)) return new HtmlWorker();
    if (['typescript', 'javascript'].includes(label)) {
      return new TypeScriptWorker();
    }
    return new EditorWorker();
  },
};

(globalThis as typeof globalThis & { MonacoEnvironment: typeof monacoEnvironment })
  .MonacoEnvironment = monacoEnvironment;
loader.config({ monaco });
