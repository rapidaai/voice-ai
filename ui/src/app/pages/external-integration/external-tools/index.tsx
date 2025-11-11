import { Helmet } from '@/app/components/Helmet';
import { ToolProviderListing } from '@/app/components/configuration/tool-provider-config/tool-provider-listing';
import { ToolProviderContextProvider } from '@/context/tool-provider-context';
import { Tab } from '@/app/components/Tab';
import { cn } from '@/styles/media';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useToolProviderPageStore } from '@/hooks/use-tool-provider-page-store';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { INTEGRATION_PROVIDER } from '@/app/components/providers';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { ExternalIntegrationProviderCard } from '@/app/components/base/cards/external-integration-provider-card';
/**
 *
 * @returns
 */
export function ToolPage() {
  const { organizationId } = useCurrentCredential();
  return (
    <div className={cn('flex flex-col h-full flex-1 overflow-auto')}>
      <Helmet title="Tools and external integrations" />
      <PageHeaderBlock className="border-b">
        <div className="flex items-center gap-3 h-11">
          <PageTitleBlock> Tools and external integrations</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${INTEGRATION_PROVIDER.length}/${INTEGRATION_PROVIDER.length}`}
          </div>
        </div>
      </PageHeaderBlock>
      <div className={cn('space-y-4')}>
        <BluredWrapper className={cn('sticky top-0 z-1')}>
          <SearchIconInput className="bg-light-background" />
          <TablePagination
            currentPage={1}
            onChangeCurrentPage={() => {}}
            totalItem={INTEGRATION_PROVIDER.length}
            pageSize={INTEGRATION_PROVIDER.length}
            onChangePageSize={() => {}}
          />
        </BluredWrapper>

        <div
          className={cn(
            'overflow-y-auto sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4 grid p-4',
          )}
        >
          {INTEGRATION_PROVIDER.map((item, idx) => (
            <ExternalIntegrationProviderCard provider={item} key={idx} />
          ))}
        </div>
      </div>
    </div>
  );
}
