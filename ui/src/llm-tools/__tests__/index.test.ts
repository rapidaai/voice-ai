import {
  APIRequestToolDefintion,
  BUILDIN_TOOLS,
  EndOfConverstaionToolDefintion,
  EndpointToolDefintion,
  KnowledgeRetrievalToolDefintion,
  TransferCallToolDefintion,
} from '../index';

describe('llm tools catalog', () => {
  it('contains expected tool codes including mcp and excludes put_on_hold', () => {
    const codes = BUILDIN_TOOLS.map(tool => tool.code);
    expect(codes).toContain('knowledge_retrieval');
    expect(codes).toContain('api_request');
    expect(codes).toContain('endpoint');
    expect(codes).toContain('end_of_conversation');
    expect(codes).toContain('transfer_call');
    expect(codes).toContain('mcp');
    expect(codes).not.toContain('put_on_hold');
  });

  it('exports valid base tool definitions', () => {
    expect(KnowledgeRetrievalToolDefintion.name).toBe('knowledge_query');
    expect(APIRequestToolDefintion.name).toBe('api_call');
    expect(EndpointToolDefintion.name).toBe('llm_call');
    expect(EndOfConverstaionToolDefintion.name).toBe('end_conversation');
    expect(TransferCallToolDefintion.name).toBe('transfer_call');

    expect(() =>
      JSON.parse(KnowledgeRetrievalToolDefintion.parameters),
    ).not.toThrow();
    expect(() => JSON.parse(APIRequestToolDefintion.parameters)).not.toThrow();
    expect(() => JSON.parse(EndpointToolDefintion.parameters)).not.toThrow();
    expect(() =>
      JSON.parse(EndOfConverstaionToolDefintion.parameters),
    ).not.toThrow();
    expect(() =>
      JSON.parse(TransferCallToolDefintion.parameters),
    ).not.toThrow();
  });
});
