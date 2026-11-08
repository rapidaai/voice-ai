import {
  StructuredListBody,
  StructuredListCell,
  StructuredListRow,
  StructuredListWrapper,
  Tile,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';
import type {
  AgentflowField,
  AgentflowNodeConfig,
  AgentflowNodeTemplate,
} from '@/app/pages/assistant/actions/agentflow';
import type { BaseNodeCardProps } from '../types';

export type GenericNodeCardProps = BaseNodeCardProps & {
  connectorRows: unknown[];
  isCompact: boolean;
  isAction: boolean;
  nodeConfig: AgentflowNodeConfig;
  nodeFields: AgentflowField[];
  nodeOutputs: string[];
  template: AgentflowNodeTemplate;
};

export function GenericNodeCard({
  toolbar,
  handles,
  Icon,
  cardClass,
  node,
  validationPill,
  width,
  connectorRows,
  isCompact,
  isAction,
  nodeConfig,
  nodeFields,
  nodeOutputs,
  template,
}: GenericNodeCardProps) {
  const renderConnectorRows = (keyPrefix: string) => (
    <StructuredListWrapper
      aria-label={`${node.label} connections`}
      isCondensed
      isFlush
      className="!m-0 !w-full !min-w-0 !border-0"
    >
      <StructuredListBody>
        {connectorRows.map((_, index) => {
          const input = template.inputs[index];
          const output = nodeOutputs[index];

          return (
            <StructuredListRow
              key={`${node.id}-${keyPrefix}-connector-${index}`}
              className="!border-b-0"
            >
              <StructuredListCell
                noWrap
                className="!grid !h-9 !grid-cols-2 !items-center !gap-4 !px-7 !py-0 text-sm"
              >
                <span className="flex min-w-0 items-center gap-2 truncate text-muted ">
                  {input ? (
                    <>
                      <span className="h-2 w-2 shrink-0 rounded-full bg-layer" />
                      <span className="truncate">{input}</span>
                    </>
                  ) : null}
                </span>
                <span className="flex min-w-0 items-center justify-end gap-2 truncate text-right text-muted ">
                  {output ? (
                    <>
                      <span className="truncate">{output}</span>
                      <span className="h-2 w-2 shrink-0 rounded-full bg-primary" />
                    </>
                  ) : null}
                </span>
              </StructuredListCell>
            </StructuredListRow>
          );
        })}
      </StructuredListBody>
    </StructuredListWrapper>
  );

  return (
    <>
      {toolbar}
      {handles}
      <Tile className={cardClass} style={{ width }}>
        <div className="flex h-12 items-center gap-3 border-b border-border-subtle px-4 text-foreground ">
          <Icon size={18} className="shrink-0" />
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <div className="min-w-0 truncate text-base font-semibold">
              {node.label || template.label}
            </div>
            {validationPill}
          </div>
        </div>
        {!isAction && !isCompact && (
          <>
            <div className="mx-4 mt-4">
              <div className="mb-2 flex items-center gap-1 text-xs font-semibold uppercase text-muted ">
                Settings
                <Toggletip align="right">
                  <ToggletipButton label="Show information">
                    <Information size={12} />
                  </ToggletipButton>
                  <ToggletipContent className="normal-case">
                    key configuration values for this node.
                  </ToggletipContent>
                </Toggletip>
              </div>
              <StructuredListWrapper
                aria-label={`${node.label} settings`}
                isCondensed
                isFlush
                className="!m-0 !w-full !min-w-0 !border border-border-subtle "
              >
                <StructuredListBody>
                  {nodeFields.map(field => (
                    <StructuredListRow
                      key={`${node.id}-field-${field.name}`}
                      className="!border-b border-border-subtle last:!border-b-0 "
                    >
                      <StructuredListCell
                        noWrap
                        className="!flex !h-10 !items-center !justify-between !gap-3 !px-3 !py-0 text-sm"
                      >
                        <span className="min-w-0 truncate text-muted">
                          {field.label}
                        </span>
                        <span className="max-w-[176px] truncate text-right text-foreground ">
                          {String(
                            nodeConfig[field.name] ?? field.defaultValue,
                          ) || 'empty'}
                        </span>
                      </StructuredListCell>
                    </StructuredListRow>
                  ))}
                </StructuredListBody>
              </StructuredListWrapper>
            </div>
            <div className="mt-4 border-t border-border-subtle pb-4 ">
              <div className="flex h-12 items-center justify-between border-b border-border-subtle px-4 ">
                <div className="flex items-center gap-1 text-xs font-semibold uppercase text-muted ">
                  Connections
                  <Toggletip align="right">
                    <ToggletipButton label="Show information">
                      <Information size={12} />
                    </ToggletipButton>
                    <ToggletipContent className="normal-case">
                      available input and output paths for this node.
                    </ToggletipContent>
                  </Toggletip>
                </div>
              </div>
              {renderConnectorRows('connector')}
            </div>
          </>
        )}
        {isCompact &&
          !isAction &&
          node.type !== 'chat-output' &&
          connectorRows.length > 0 && (
            <div className="pb-4">{renderConnectorRows('compact')}</div>
          )}
      </Tile>
    </>
  );
}
