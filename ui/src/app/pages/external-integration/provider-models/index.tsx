import { Helmet } from '@/app/components/app-shell/helmet';
import { ProviderCard } from '@/app/components/domain/cards/provider-card';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import { Text } from '@/app/components/ui/primitives';
import { cn } from '@/utils';
import { INTEGRATION_PROVIDER } from '@/providers';
import { useState } from 'react';
import {
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
} from '@carbon/react';
/**
 *
 * @returns
 */
export function ProviderModelPage() {
  const [searchTerm, setSearchTerm] = useState(''); // State to store search input
  const filteredProviders = INTEGRATION_PROVIDER.filter(provider =>
    provider.name.toLowerCase().includes(searchTerm.toLowerCase()),
  );

  return (
    <div className={cn('flex flex-col h-full flex-1 overflow-auto')}>
      <Helmet title="Providers and Models" />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Integrations
          </Text>
        }
        title="Providers and Models"
        description="Browse the model, voice, transcription, and platform providers available to your workspace."
        contextualActions={
          <Text variant="label-01" className="text-muted tabular-nums">
            {filteredProviders.length} of {INTEGRATION_PROVIDER.length}
          </Text>
        }
      />
      <PageHeaderStickyToolbar>
        <TableToolbar>
          <TableToolbarContent>
            <TableToolbarSearch
              placeholder="Search providers..."
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
            />
          </TableToolbarContent>
        </TableToolbar>
      </PageHeaderStickyToolbar>
      <section className="grid content-start grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4 grow shrink-0 m-4">
        {filteredProviders.map((mp, idx) => {
          return (
            <ProviderCard
              key={`spc-${idx}`}
              provider={mp}
              className="col-span-1"
            />
          );
        })}
      </section>
    </div>
  );
}
