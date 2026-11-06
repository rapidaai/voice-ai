import { Helmet } from '@/app/components/app-shell/helmet';
import { ProviderCard } from '@/app/components/domain/cards/provider-card';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/layout/blocks/page-title-block';
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
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Providers and Models</PageTitleBlock>
          <span className="text-xs text-muted tabular-nums">
            {`${filteredProviders.length}/${INTEGRATION_PROVIDER.length}`}
          </span>
        </div>
      </PageHeaderBlock>
      <TableToolbar>
        <TableToolbarContent>
          <TableToolbarSearch
            placeholder="Search providers..."
            value={searchTerm}
            onChange={e => setSearchTerm(e.target.value)}
          />
        </TableToolbarContent>
      </TableToolbar>
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
