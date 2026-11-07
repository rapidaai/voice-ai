import { Select, SelectItem } from '@carbon/react';
import { TextInput, TextArea, Stack } from '@/app/components/ui/primitives';
import { ConfigureToolProps, useParameterManager } from './common';
import { BlueNoticeBlock } from '@/app/components/layout/container/message/notice-block';
import { APiStringHeader } from '@/app/components/domain/external-api/api-header';
import { MCP_PROTOCOL_OPTIONS } from './mcp/constant';
import { InputGroup } from '@/app/components/ui/primitives';

// ============================================================================
// Main Component
// ============================================================================

export function ConfigureMCP({
  toolDefinition,
  onChangeToolDefinition,
  onParameterChange,
  parameters,
  inputClass,
}: ConfigureToolProps) {
  const { getParamValue, updateParameter } = useParameterManager(
    parameters,
    onParameterChange,
  );

  const serverUrl = getParamValue('mcp.server_url');
  const protocol = getParamValue('mcp.protocol') || 'sse';
  const timeout = getParamValue('mcp.timeout') || '30';
  const headers = getParamValue('mcp.headers');

  const handleChange = (field: 'name' | 'description', value: string) => {
    if (toolDefinition && onChangeToolDefinition) {
      onChangeToolDefinition({ ...toolDefinition, [field]: value });
    }
  };

  return (
    <>
      <InputGroup title="Action Definition">
        <Stack gap={7}>
          <TextInput
            id="mcp-tool-name"
            labelText="Name"
            value={toolDefinition?.name || ''}
            onChange={e => handleChange('name', e.target.value)}
            placeholder="Enter MCP tool name"
          />

          <TextArea
            id="mcp-tool-description"
            labelText="Description"
            value={toolDefinition?.description || ''}
            onChange={e => handleChange('description', e.target.value)}
            placeholder="A tool description or definition of when this MCP tool will get triggered."
            rows={2}
          />

          <TextInput
            id="mcp-server-url"
            labelText="MCP Server URL"
            value={serverUrl}
            onChange={e => updateParameter('mcp.server_url', e.target.value)}
            placeholder="https://your-mcp-server.com"
            type="url"
          />

          <div className="grid grid-cols-2 gap-4">
            <Select
              id="mcp-protocol"
              labelText="Protocol"
              value={protocol}
              onChange={e => updateParameter('mcp.protocol', e.target.value)}
            >
              {MCP_PROTOCOL_OPTIONS.map(o => (
                <SelectItem key={o.value} value={o.value} text={o.name} />
              ))}
            </Select>

            <TextInput
              id="mcp-timeout"
              labelText="Timeout (seconds)"
              value={timeout}
              onChange={e => updateParameter('mcp.timeout', e.target.value)}
              placeholder="30"
              type="number"
            />
          </div>

          <div>
            <p className="text-xs font-medium mb-2">Headers</p>
            <APiStringHeader
              inputClass={inputClass}
              headerValue={headers}
              setHeaderValue={value => updateParameter('mcp.headers', value)}
            />
          </div>

          <BlueNoticeBlock>
            <div className="text-sm text-support-info ">
              <div className="text-support-info ">
                This tool will proxy calls to the specified MCP server. If you
                provide a specific MCP Tool Name, it will call that tool on the
                server; otherwise, it will use the tool name specified above.
                The LLM will see the name and description you provide above.
              </div>
            </div>
          </BlueNoticeBlock>
        </Stack>
      </InputGroup>
    </>
  );
}
