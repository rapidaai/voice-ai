import React, { FC } from 'react';
import { Endpoint } from '@rapidaai/react';
import { useEndpointPageStore } from '@/hooks';
import { TagColumn } from '@/app/components/Table/TagColumn';

import { TickIcon } from '@/app/components/Icon/Tick';
import { cn, toHumanReadableRelativeTime } from '@/styles/media';
import { CopyableColumn } from '@/app/components/Table/CopyableColumn';
import { LabelColumn } from '@/app/components/Table/LabelColumn';
import { DateColumn } from '@/app/components/Table/DateColumn';
import { CostColumn } from '@/app/components/Table/CostColumn';
import { NumberColumn } from '@/app/components/Table/NumberColumn';
import { LatencyColumn } from '@/app/components/Table/LatencyColumn';
import { TextColumn } from '@/app/components/Table/TextColumn';
import { TableRow } from '@/app/components/base/tables/table-row';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { ProviderPill } from '@/app/components/Pill/provider-model-pill';
import { nanoToMilli } from '@/utils';
import { CustomLink } from '@/app/components/custom-link';

/**
 *
 */
interface SingleEndpointProps {
  /**
   * current endpoint
   */
  endpoint: Endpoint;
}

/**
 *
 * @param props
 * @returns
 */
export const SingleEndpoint: FC<SingleEndpointProps> = ({ endpoint }) => {
  const endpointAction = useEndpointPageStore();

  const getErrorRate = (endpoint: Endpoint) => {
    const errorCount = parseInt(
      endpoint.getEndpointanalytics()?.getErrorcount() ?? '0',
      10,
    );
    const totalCount = parseInt(
      endpoint.getEndpointanalytics()?.getCount() ?? '0',
      10,
    );
    if (errorCount === 0 || totalCount === 0) {
      return 0;
    }

    return Number((errorCount / totalCount) * 100).toFixed(2);
  };
  return (
    <TableRow>
      {endpointAction.visibleColumn('getStatus') && (
        <TableCell className="min-w-60">
          <div className="flex items-center space-x-1.5">
            <span className="p-1 bg-green-400/20 text-green-600 rounded-[2px] w-fit block">
              <TickIcon className="w-6 h-6" />
            </span>
            <div>
              <span className="text-green-600 font-medium block leading-3">
                Deployed
              </span>
              <span className="opacity-60 text-xs leading-3">
                Deployed{' '}
                {endpoint.getEndpointprovidermodel()?.getCreateddate() &&
                  toHumanReadableRelativeTime(
                    endpoint.getEndpointprovidermodel()?.getCreateddate()!,
                  )}
              </span>
            </div>
          </div>
        </TableCell>
      )}
      {endpointAction.visibleColumn('getName') && (
        <TableCell>
          <CustomLink
            to={`/deployment/endpoint/${endpoint.getId()}`}
            className="text-blue-600 underline"
          >
            {endpoint?.getName()}
          </CustomLink>
        </TableCell>
      )}
      {endpointAction.visibleColumn('getId') && (
        <TableCell>{endpoint.getId()}</TableCell>
      )}
      {endpointAction.visibleColumn('getId') && (
        <TableCell>{endpoint.getId()}</TableCell>
      )}
      {endpointAction.visibleColumn('getVersion') && (
        <CopyableColumn>{`vrsn_${endpoint
          .getEndpointprovidermodel()
          ?.getId()}`}</CopyableColumn>
      )}
      {endpointAction.visibleColumn('getTags') && (
        <TagColumn tags={endpoint.getEndpointtag()?.getTagList()} />
      )}
      {endpointAction.visibleColumn('getCount') && (
        <LabelColumn className="bg-blue-300/10 text-blue-500 dark:text-blue-400 ">
          {endpoint.getEndpointanalytics()?.getCount()}
        </LabelColumn>
      )}
      {endpointAction.visibleColumn('getErrorRate') && (
        <LabelColumn className="bg-red-300/10 text-red-500 dark:text-red-400 ">
          {getErrorRate(endpoint)}%
        </LabelColumn>
      )}
      {endpointAction.visibleColumn('getCurrentModel') && (
        <TableCell className="min-w-60">
          <ProviderPill
            providerId={endpoint
              .getEndpointprovidermodel()
              ?.getModelproviderid()}
          />
        </TableCell>
      )}
      {endpointAction.visibleColumn('getCost') && (
        <CostColumn
          cost={
            endpoint.getEndpointanalytics()?.getTotalinputcost()! +
            endpoint.getEndpointanalytics()?.getTotaloutputcost()!
          }
        />
      )}
      {endpointAction.visibleColumn('getTotalToken') && (
        <NumberColumn num={endpoint.getEndpointanalytics()?.getTotaltoken()} />
      )}
      {endpointAction.visibleColumn('getP50') && (
        <LatencyColumn>
          {nanoToMilli(endpoint.getEndpointanalytics()?.getP50latency())}
        </LatencyColumn>
      )}
      {endpointAction.visibleColumn('getP99') && (
        <LatencyColumn>
          {nanoToMilli(endpoint.getEndpointanalytics()?.getP99latency())}
        </LatencyColumn>
      )}
      {endpointAction.visibleColumn('getMRR') &&
        (endpoint.getEndpointanalytics()?.getLastactivity() &&
        endpoint
          .getEndpointanalytics()
          ?.getLastactivity()
          ?.toDate()
          .getTime()! > new Date('1970-01-01').getTime() ? (
          <DateColumn
            date={endpoint.getEndpointanalytics()?.getLastactivity()}
          />
        ) : (
          <TextColumn className="opacity-50">No yet run</TextColumn>
        ))}
      {endpointAction.visibleColumn('getCreatedBy') && (
        <TableCell>
          <span
            className={cn(
              'font-medium hover:underline hover:text-blue-600 outline-offset-4 cursor-pointer',
              'capitalize',
            )}
          >
            {endpoint.getEndpointprovidermodel()?.getCreateduser()?.getName()}
          </span>
        </TableCell>
      )}
    </TableRow>
  );
};
