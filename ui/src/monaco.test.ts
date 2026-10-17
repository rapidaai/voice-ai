const mockLoaderConfig = jest.fn();
const mockLanguageLoads: string[] = [];
const mockMonaco = { editor: {}, languages: {} };
const mockEditorWorkerClass = class EditorWorker {};
const mockJsonWorkerClass = class JsonWorker {};
const mockCssWorkerClass = class CssWorker {};
const mockHtmlWorkerClass = class HtmlWorker {};
const mockTypeScriptWorkerClass = class TypeScriptWorker {};

jest.mock('@monaco-editor/react', () => ({
  loader: { config: mockLoaderConfig },
}));
jest.mock('monaco-editor/esm/vs/editor/editor.api', () => ({
  __esModule: true,
  ...mockMonaco,
}));
jest.mock(
  'monaco-editor/esm/vs/basic-languages/html/html.contribution',
  () => mockLanguageLoads.push('html'),
);
jest.mock(
  'monaco-editor/esm/vs/basic-languages/javascript/javascript.contribution',
  () => mockLanguageLoads.push('javascript'),
);
jest.mock(
  'monaco-editor/esm/vs/basic-languages/twig/twig.contribution',
  () => mockLanguageLoads.push('twig'),
);
jest.mock(
  'monaco-editor/esm/vs/basic-languages/typescript/typescript.contribution',
  () => mockLanguageLoads.push('typescript'),
);
jest.mock(
  'monaco-editor/esm/vs/language/html/monaco.contribution',
  () => mockLanguageLoads.push('html-service'),
);
jest.mock(
  'monaco-editor/esm/vs/language/json/monaco.contribution',
  () => mockLanguageLoads.push('json-service'),
);
jest.mock(
  'monaco-editor/esm/vs/language/typescript/monaco.contribution',
  () => mockLanguageLoads.push('typescript-service'),
);
jest.mock('./monaco-workers', () => ({
  CssWorker: mockCssWorkerClass,
  EditorWorker: mockEditorWorkerClass,
  HtmlWorker: mockHtmlWorkerClass,
  JsonWorker: mockJsonWorkerClass,
  TypeScriptWorker: mockTypeScriptWorkerClass,
}));

require('./monaco');

const monacoEnvironment = (
  globalThis as typeof globalThis & {
    MonacoEnvironment: {
      getWorker(moduleId: string, label: string): Worker;
    };
  }
).MonacoEnvironment;

describe('Monaco setup', () => {
  it('registers the configured editor instance and supported languages', () => {
    expect(mockLoaderConfig).toHaveBeenCalledWith({
      monaco: expect.objectContaining(mockMonaco),
    });
    expect(mockLanguageLoads).toEqual([
      'html',
      'javascript',
      'twig',
      'typescript',
      'html-service',
      'json-service',
      'typescript-service',
    ]);
  });

  it('routes language workers and falls back to the editor worker', () => {
    expect(monacoEnvironment.getWorker('', 'json')).toBeInstanceOf(
      mockJsonWorkerClass,
    );
    expect(monacoEnvironment.getWorker('', 'scss')).toBeInstanceOf(
      mockCssWorkerClass,
    );
    expect(monacoEnvironment.getWorker('', 'html')).toBeInstanceOf(
      mockHtmlWorkerClass,
    );
    expect(monacoEnvironment.getWorker('', 'javascript')).toBeInstanceOf(
      mockTypeScriptWorkerClass,
    );
    expect(monacoEnvironment.getWorker('', 'unknown')).toBeInstanceOf(
      mockEditorWorkerClass,
    );
  });
});
