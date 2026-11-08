import { FC } from 'react';
import { Endpoint } from '@rapidaai/react';
import { useEndpointPageStore } from '@/stores/endpoint';
import { nanoToMilli, toHumanReadableRelativeTime } from '@/utils/date';
import { useNavigate } from 'react-router-dom';
import { TableRow, TableCell, Tag, Link } from '@carbon/react';
import { ProviderTag } from '@/app/components/domain/provider-tag';
import { Launch, View, SourceControl } from '@carbon/icons-react';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
import { VersionIndicator } from '@/app/components/domain/indicators/version';
import { IconOnlyButton } from '@/app/components/ui/primitives';
import { CopyButton } from '@/app/components/ui/primitives';
import { cn } from '@/utils';
import { auditActorLabel, createdAuditActor } from '@/utils/audit-actor';

interface SingleEndpointProps {
  endpoint: Endpoint;
}

export const SingleEndpoint: FC<SingleEndpointProps> = ({ endpoint }) => {
  const endpointAction = useEndpointPageStore();
  const navigate = useNavigate();
  const endpointId = endpoint.getId();
  const analytics = endpoint.getEndpointanalytics();
  const providerModel = endpoint.getEndpointprovidermodel();
  const status =
    endpoint.getStatus() || providerModel?.getStatus() || 'DEPLOYED';
  const errorRate = getErrorRate(endpoint);
  const tags = endpoint.getEndpointtag()?.getTagList() ?? [];
  const visibleTags = tags.slice(0, 2);
  const overflowTagCount = Math.max(tags.length - visibleTags.length, 0);
  const totalCost =
    (analytics?.getTotalinputcost() ?? 0) +
    (analytics?.getTotaloutputcost() ?? 0);
  const lastActivity = analytics?.getLastactivity();

  return (
    <TableRow>
      {endpointAction.visibleColumn('getName') && (
        <TableCell className="text-sm">
          <div className="flex min-w-0 items-center gap-1">
            <Link
              href={`/deployment/endpoint/${endpointId}`}
              className="!inline-flex !min-w-0 !items-center !gap-1 !text-sm"
            >
              <span className="truncate">{endpoint.getName()}</span>
              <Launch size={12} className="shrink-0" />
            </Link>
          </div>
        </TableCell>
      )}
      {endpointAction.visibleColumn('getId') && (
        <TableCell className="max-w-[260px]">
          <div className="flex min-w-0 items-center gap-1">
            <span className="truncate font-mono text-[13px]">
              {endpointId || '-'}
            </span>
            {endpointId && (
              <CopyButton className="h-6 w-6 shrink-0">{endpointId}</CopyButton>
            )}
          </div>
        </TableCell>
      )}
      {endpointAction.visibleColumn('getStatus') && (
        <TableCell className="text-sm">
          <RecordStatusIndicator state={status} />
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCurrentModel') && (
        <TableCell className="text-sm">
          <ProviderTag provider={providerModel?.getModelprovidername()} />
        </TableCell>
      )}
      {endpointAction.visibleColumn('getVersion') && (
        <TableCell className="text-sm">
          {providerModel?.getId() ? (
            <VersionIndicator id={providerModel.getId()} />
          ) : (
            <span className="text-sm text-muted">-</span>
          )}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getTags') && (
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
            <span className="text-sm text-muted">-</span>
          )}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCount') && (
        <TableCell className="min-w-28 whitespace-nowrap text-sm tabular-nums">
          {formatInteger(parseMetric(analytics?.getCount()))}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getErrorRate') && (
        <TableCell className="min-w-36 whitespace-nowrap text-sm">
          <span className={cn('tabular-nums', errorRateClass(errorRate))}>
            {formatPercent(errorRate)}
          </span>
        </TableCell>
      )}
      {endpointAction.visibleColumn('getP50') && (
        <TableCell className="min-w-32 whitespace-nowrap font-mono text-[13px] tabular-nums">
          {formatLatency(analytics?.getP50latency())}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getP99') && (
        <TableCell className="min-w-32 whitespace-nowrap font-mono text-[13px] tabular-nums">
          {formatLatency(analytics?.getP99latency())}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCost') && (
        <TableCell className="min-w-28 whitespace-nowrap font-mono text-[13px] tabular-nums">
          {formatCurrency(totalCost)}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getMRR') && (
        <TableCell className="min-w-36 whitespace-nowrap text-[13px]">
          {lastActivity &&
          lastActivity.toDate().getTime() > new Date('1970-01-01').getTime() ? (
            toHumanReadableRelativeTime(lastActivity)
          ) : (
            <span className="text-muted">Not yet run</span>
          )}
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCreatedActor') && (
        <TableCell className="text-sm">
          <span className="capitalize">
            {auditActorLabel(createdAuditActor(providerModel))}
          </span>
        </TableCell>
      )}
      {endpointAction.visibleColumn('action') && (
        <TableCell>
          <div className="flex items-center gap-0">
            <IconOnlyButton
              kind="ghost"
              size="md"
              renderIcon={View}
              iconDescription="View detail"
              onClick={() => navigate(`/deployment/endpoint/${endpointId}`)}
            />
            <IconOnlyButton
              kind="ghost"
              size="md"
              renderIcon={SourceControl}
              iconDescription="Create new version"
              onClick={() =>
                navigate(
                  `/deployment/endpoint/${endpointId}/create-endpoint-version`,
                )
              }
            />
          </div>
        </TableCell>
      )}
    </TableRow>
  );
};

const parseMetric = (value: string | number | undefined): number => {
  const metric = Number(value ?? 0);
  return Number.isFinite(metric) ? metric : 0;
};

const formatInteger = (value: number): string =>
  new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 }).format(value);

const formatCurrency = (value: number): string =>
  new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 4,
    maximumFractionDigits: 4,
  }).format(value);

const formatLatency = (value: number | string | undefined): string => {
  const latency = nanoToMilli(value);
  return latency === undefined ? '-' : `${latency}ms`;
};

const getErrorRate = (endpoint: Endpoint): number => {
  const errorCount = parseMetric(
    endpoint.getEndpointanalytics()?.getErrorcount(),
  );
  const totalCount = parseMetric(endpoint.getEndpointanalytics()?.getCount());
  if (errorCount === 0 || totalCount === 0) return 0;
  return Number(((errorCount / totalCount) * 100).toFixed(2));
};

const formatPercent = (value: number): string =>
  `${Number.isInteger(value) ? value : value.toFixed(2)}%`;

const errorRateClass = (errorRate: number): string => {
  if (errorRate === 0) return 'text-support-success ';
  if (errorRate < 5) return 'text-support-warning ';
  return 'text-support-error ';
};
