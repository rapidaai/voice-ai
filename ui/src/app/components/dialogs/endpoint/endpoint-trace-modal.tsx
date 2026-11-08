import { useState } from 'react';
import type { EndpointLog } from '@rapidaai/react';
import { RightSideModal } from '@/app/components/dialogs/shared';
import type { ModalProps } from '@/app/components/ui/primitives';
import { SourceIndicator } from '@/app/components/domain/indicators/source';
import { EndpointArguments } from './endpoint-trace-arguments';
import { EndpointMetadatas } from './endpoint-trace-metadatas';
import { EndpointMetrics } from './endpoint-trace-metrics';
import { EndpointOptions } from './endpoint-trace-options';
import { toHumanReadableDateTime } from '@/utils/date';
import { getTotalTokenMetric } from '@/utils/metadata';
import { OverviewRow } from '@/app/components/dialogs/shared';
import { Tabs } from '@/app/components/ui/primitives';
import { CarbonStatusIndicator } from '@/app/components/ui/feedback';

interface EndpointTraceModalProps extends ModalProps {
  currentTrace: EndpointLog | null;
}

export function EndpointTraceModal({
  modalOpen,
  setModalOpen,
  currentTrace,
}: EndpointTraceModalProps) {
  const [selectedTab, setSelectedTab] = useState(0);
  if (!currentTrace) return null;

  return (
    <RightSideModal
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
      className="w-[580px]"
      label="Trace"
      title={currentTrace.getId()}
    >
      <div className="relative flex flex-col flex-1 min-h-0">
        <Tabs
          tabs={['Overview', 'Metrics', 'Metadata', 'Options', 'Arguments']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          fill
          aria-label="Endpoint trace tabs"
          panelClassName="overflow-auto p-0"
        >
          <div className="divide-y divide-border-subtle w-full">
            <OverviewRow label="Status">
              <CarbonStatusIndicator state={currentTrace.getStatus()} />
            </OverviewRow>
            <OverviewRow label="Source">
              <SourceIndicator source={currentTrace.getSource()} />
            </OverviewRow>
            <OverviewRow label="Version">
              <span className="text-sm font-mono text-foreground ">
                vrsn_{currentTrace.getEndpointprovidermodelid()}
              </span>
            </OverviewRow>
            <OverviewRow label="Time Taken">
              <span className="text-sm tabular-nums text-foreground ">
                {Number(currentTrace.getTimetaken()) / 1000000}ms
              </span>
            </OverviewRow>
            <OverviewRow label="Total Tokens">
              <span className="text-sm tabular-nums text-foreground ">
                {getTotalTokenMetric(currentTrace.getMetricsList())}
              </span>
            </OverviewRow>
            {currentTrace.getCreateddate() && (
              <OverviewRow label="Created">
                <span className="text-sm text-foreground ">
                  {toHumanReadableDateTime(currentTrace.getCreateddate()!)}
                </span>
              </OverviewRow>
            )}
          </div>
          <EndpointMetrics metrics={currentTrace.getMetricsList()} />
          <EndpointMetadatas metadata={currentTrace.getMetadataList()} />
          <EndpointOptions options={currentTrace.getOptionsList()} />
          <EndpointArguments args={currentTrace.getArgumentsList()} />
        </Tabs>
      </div>
    </RightSideModal>
  );
}
