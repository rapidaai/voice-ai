import React, { FC } from 'react';
import { Assistant } from '@rapidaai/react';
import { toHumanReadableDateTime } from '@/utils/date';
import SourceIndicator from '@/app/components/indicators/source';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { Launch, Rocket, SourceControl, View } from '@carbon/icons-react';
import { Link, TableRow, TableCell, Tag } from '@carbon/react';
import { RecordStatusIndicator } from '@/app/components/carbon/record-status-indicator';
import { IconOnlyButton } from '@/app/components/carbon/button';
import { CopyButton } from '@/app/components/carbon/button/copy-button';
import { VersionIndicator } from '@/app/components/indicators/version';

const SingleAssistant: FC<{ assistant: Assistant }> = ({ assistant }) => {
  const gn = useGlobalNavigation();
  const assistantId = assistant.getId();
  const status = assistant.getStatus();
  const tags = assistant.getAssistanttag()?.getTagList() ?? [];
  const visibleTags = tags.slice(0, 2);
  const overflowTagCount = Math.max(tags.length - visibleTags.length, 0);
  const owner = assistant.getCreateduser()?.getName();
  const hasDeployment = hasAssistantDeployment(assistant);

  return (
    <TableRow>
      <TableCell className="text-sm">
        <Link
          href={`/deployment/assistant/${assistantId}`}
          className="!inline-flex !min-w-0 !items-center !gap-1 !text-sm"
        >
          <span className="truncate">{assistant.getName()}</span>
          <Launch size={12} className="shrink-0" />
        </Link>
      </TableCell>

      <TableCell className="max-w-[260px]">
        <div className="flex min-w-0 items-center gap-1">
          <span className="truncate font-mono text-[13px]">
            {assistantId || '-'}
          </span>
          {assistantId && (
            <CopyButton className="h-6 w-6 shrink-0">{assistantId}</CopyButton>
          )}
        </div>
      </TableCell>

      <TableCell className="text-sm">
        <span className="capitalize">{formatProvider(assistant)}</span>
      </TableCell>

      <TableCell className="text-sm">
        {assistant.getAssistantproviderid() ? (
          <VersionIndicator id={assistant.getAssistantproviderid()} />
        ) : (
          <span className="text-sm text-gray-400">-</span>
        )}
      </TableCell>

      <TableCell className="text-sm">
        {status ? (
          <RecordStatusIndicator state={status} />
        ) : (
          <span className="text-sm text-gray-400">-</span>
        )}
      </TableCell>

      <TableCell className="text-sm">
        {hasDeployment ? (
          <div className="flex flex-wrap gap-1">
            {assistant.getApideployment() && (
              <SourceIndicator source="react-sdk" withLabel={false} />
            )}
            {assistant.getDebuggerdeployment() && (
              <SourceIndicator source="debugger" withLabel={false} />
            )}
            {assistant.getWebplugindeployment() && (
              <SourceIndicator source="web-plugin" withLabel={false} />
            )}
            {assistant.getPhonedeployment() && (
              <SourceIndicator source="twilio-call" withLabel={false} />
            )}
          </div>
        ) : (
          <span className="text-sm text-gray-400">Not configured</span>
        )}
      </TableCell>

      <TableCell>
        <div className="flex items-center gap-0">
          <IconOnlyButton
            kind="ghost"
            size="md"
            renderIcon={View}
            iconDescription="View detail"
            onClick={() => gn.goToAssistant(assistantId)}
          />
          {hasDeployment ? (
            <IconOnlyButton
              kind="ghost"
              size="md"
              renderIcon={Rocket}
              iconDescription="Manage deployments"
              onClick={() => gn.goToManageAssistant(assistantId)}
            />
          ) : (
            <IconOnlyButton
              kind="ghost"
              size="md"
              renderIcon={Rocket}
              iconDescription="Set up deployment"
              onClick={() => gn.goToManageAssistant(assistantId)}
            />
          )}
          <IconOnlyButton
            kind="ghost"
            size="md"
            renderIcon={SourceControl}
            iconDescription="Create new version"
            onClick={() => gn.goToCreateAssistantVersion(assistantId)}
          />
        </div>
      </TableCell>

      <TableCell className="text-sm">
        {tags.length > 0 ? (
          <div className="flex flex-wrap gap-1">
            {visibleTags.map(tag => (
              <Tag key={tag} type="cool-gray" size="sm">
                {tag}
              </Tag>
            ))}
            {overflowTagCount > 0 && (
              <Tag type="gray" size="sm">
                +{overflowTagCount}
              </Tag>
            )}
          </div>
        ) : (
          <span className="text-sm text-gray-400">-</span>
        )}
      </TableCell>

      <TableCell className="min-w-36 whitespace-nowrap text-[13px]">
        {assistant.getUpdateddate()
          ? toHumanReadableDateTime(assistant.getUpdateddate()!)
          : '-'}
      </TableCell>

      <TableCell className="text-sm">
        <span className="capitalize">{owner || '-'}</span>
      </TableCell>
    </TableRow>
  );
};

const hasAssistantDeployment = (assistant: Assistant): boolean =>
  Boolean(
    assistant.getApideployment() ||
      assistant.getDebuggerdeployment() ||
      assistant.getWebplugindeployment() ||
      assistant.getPhonedeployment(),
  );

const formatProvider = (assistant: Assistant): string => {
  const provider = assistant.getAssistantprovider();
  if (provider) return provider.replace(/[-_]/g, ' ').toLowerCase();
  if (assistant.getAssistantprovidermodel()) return 'prompt';
  if (assistant.getAssistantprovideragentkit()) return 'agentkit';
  if (assistant.getAssistantproviderwebsocket()) return 'websocket';
  if (assistant.getAssistantprovideragentflow()) return 'agentflow';
  return '-';
};

export default React.memo(SingleAssistant);
