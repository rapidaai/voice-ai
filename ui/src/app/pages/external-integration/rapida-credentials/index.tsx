import { useState, useContext, useEffect, useRef } from 'react';
import { useCredential, useCurrentCredential } from '@/hooks/use-credential';
import {
  GetAllProjectCredentialResponse,
  ProjectCredential,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/app-shell/helmet';
import { EmptyState } from '@/app/components/ui/feedback';
import { AuthContext } from '@/context/auth-context';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import {
  Add,
  ApiKey,
  Checkmark,
  Copy,
  Renew,
  Unlink,
  View,
  ViewOff,
} from '@carbon/icons-react';
import {
  DataTable,
  DataTableSkeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
  Tag,
} from '@carbon/react';
import { toHumanReadableDateTime } from '@/utils/date';
import { DocNoticeBlock } from '@/app/components/layout/container/message/notice-block/doc-notice-block';
import {
  BaseCard,
  CopyButton,
  FieldSet,
  FormLabel,
  GhostButton,
  IconOnlyButton,
  Pagination,
  PrimaryButton,
  Text,
  typographyClassNames,
} from '@/app/components/ui/primitives';
import {
  ScrollableTableSection,
  TableSection,
} from '@/app/components/layout/sections/table-section';
import { ConfirmDeleteDialog } from '@/app/components/dialogs/shared';
import { listProjectCredentials, revokeProjectCredential } from '@/clients';
import { CreateProjectToken } from '@/app/components/domain/project-credentials/create-project-token';
import {
  ProjectAPIKeyQuerySearch,
  type ProjectAPIKeySearchCriteria,
} from './project-api-key-query-search';

const projectAPIKeyHeaders = [
  { key: 'name', header: 'Name' },
  { key: 'apiKey', header: 'API key' },
  { key: 'role', header: 'Access role' },
  { key: 'created', header: 'Created' },
  { key: 'actions', header: 'Actions' },
];

export function ProjectCredentialPage() {
  const [credentials, setCredentials] = useState<ProjectCredential[]>([]);
  const [querySearchValue, setQuerySearchValue] = useState('');
  const [criteria, setCriteria] = useState<ProjectAPIKeySearchCriteria[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [totalItems, setTotalItems] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [hasCompletedInitialLoad, setHasCompletedInitialLoad] = useState(false);
  const [error, setError] = useState('');
  const [refresh, setRefresh] = useState(0);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [visibleCredentialIds, setVisibleCredentialIds] = useState<Set<string>>(
    new Set(),
  );
  const [copiedCredentialId, setCopiedCredentialId] = useState<string | null>(
    null,
  );
  const [credentialPendingRevoke, setCredentialPendingRevoke] =
    useState<ProjectCredential | null>(null);
  const requestSequence = useRef(0);
  const { getCurrentProject } = useContext(AuthContext);
  const currentProject = getCurrentProject?.();
  const [userId, token, , organizationId] = useCredential();

  useEffect(() => {
    if (!currentProject?.id) return;
    const requestId = ++requestSequence.current;
    setIsLoading(true);
    setError('');
    listProjectCredentials({
      projectId: currentProject.id,
      page,
      pageSize,
      criteria,
      auth: { organizationId, token, userId },
      callback: (
        requestError,
        response: GetAllProjectCredentialResponse | null,
      ) => {
        if (requestId !== requestSequence.current) return;
        setIsLoading(false);
        setHasCompletedInitialLoad(true);
        if (requestError || !response?.getSuccess()) {
          setError('Try again. No API keys were changed.');
          toast.error('API keys could not be refreshed. Try again.');
          return;
        }
        setCredentials(response.getDataList());
        setTotalItems(response.getPaginated()?.getTotalitem() || 0);
      },
    });
    return () => {
      requestSequence.current += 1;
    };
  }, [criteria, currentProject, page, pageSize, refresh, token, userId]);

  const rows = credentials.map(credential => ({
    id: credential.getId(),
    name: credential.getName(),
    apiKey: '',
    role: credential.getRolename(),
    created: credential.getCreateddate()
      ? toHumanReadableDateTime(credential.getCreateddate()!)
      : '-',
    actions: '',
  }));

  return (
    <div
      data-page-scroll
      className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden bg-surface"
    >
      <Helmet title="Project API keys" />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Integrations
          </Text>
        }
        title="Project API keys"
        description="Manage API keys used by applications and tools to access this project."
        contextualActions={
          <Text variant="label-01" className="text-muted tabular-nums">
            {credentials.length} of {totalItems}
          </Text>
        }
        pageActions={
          <PrimaryButton
            className="w-full md:w-auto"
            size="md"
            renderIcon={Add}
            onClick={() => setIsCreateOpen(true)}
          >
            Create API key
          </PrimaryButton>
        }
      />
      {isLoading && !hasCompletedInitialLoad ? (
        <TableSection className="min-h-0 overflow-hidden bg-surface">
          <DataTableSkeleton
            headers={projectAPIKeyHeaders}
            columnCount={projectAPIKeyHeaders.length}
            rowCount={5}
            showHeader={false}
            showToolbar
            size="lg"
            className="w-full"
          />
        </TableSection>
      ) : (
        <TableSection className="min-h-0 bg-surface">
          <PageHeaderStickyToolbar>
            <TableToolbar>
              <TableToolbarContent className="flex-col items-stretch md:flex-row md:items-center">
                <ProjectAPIKeyQuerySearch
                  value={querySearchValue}
                  onChange={setQuerySearchValue}
                  onApply={nextCriteria => {
                    setPage(1);
                    setCriteria(nextCriteria);
                  }}
                />
                <IconOnlyButton
                  size="md"
                  renderIcon={Renew}
                  iconDescription="Refresh API keys"
                  kind="ghost"
                  tooltipPosition="bottom"
                  onClick={() => setRefresh(value => value + 1)}
                />
              </TableToolbarContent>
            </TableToolbar>
          </PageHeaderStickyToolbar>

          {error && credentials.length === 0 ? (
            <EmptyState
              icon={ApiKey}
              title="API keys could not be loaded"
              subtitle={error}
              action="Try again"
              actionIcon={Renew}
              onAction={() => setRefresh(value => value + 1)}
            />
          ) : rows.length === 0 ? (
            <EmptyState
              icon={ApiKey}
              title={
                criteria.length > 0
                  ? 'No API keys match your search.'
                  : 'No project API keys'
              }
              subtitle={
                criteria.length > 0
                  ? undefined
                  : 'Create an API key for an application or tool that needs access to this project.'
              }
              action={criteria.length > 0 ? undefined : 'Create API key'}
              actionIcon={criteria.length > 0 ? undefined : Add}
              onAction={
                criteria.length > 0 ? undefined : () => setIsCreateOpen(true)
              }
            />
          ) : (
            <>
              <ScrollableTableSection className="shrink-0">
                <DataTable rows={rows} headers={projectAPIKeyHeaders}>
                  {({
                    rows: tableRows,
                    headers,
                    getHeaderProps,
                    getRowProps,
                    getTableProps,
                  }) => (
                    <Table {...getTableProps()} className="min-w-[60rem]">
                      <TableHead>
                        <TableRow>
                          {headers.map(header => (
                            <TableHeader
                              {...getHeaderProps({ header })}
                              key={header.key}
                              className="whitespace-nowrap"
                            >
                              {header.header}
                            </TableHeader>
                          ))}
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {tableRows.map(row => {
                          const credential = credentials.find(
                            current => current.getId() === row.id,
                          );
                          if (!credential) return null;
                          const isVisible = visibleCredentialIds.has(row.id);
                          return (
                            <TableRow {...getRowProps({ row })} key={row.id}>
                              {row.cells.map(cell => (
                                <TableCell
                                  key={cell.id}
                                  className={
                                    typographyClassNames['body-compact-01']
                                  }
                                >
                                  {cell.info.header === 'name' ? (
                                    credential.getName()
                                  ) : cell.info.header === 'apiKey' ? (
                                    <div className="flex w-80 items-center gap-1">
                                      <Text
                                        as="code"
                                        variant="code-02"
                                        id={`project-api-key-${row.id}`}
                                        className="w-64 shrink-0 truncate"
                                      >
                                        {isVisible
                                          ? credential.getKey()
                                          : `${credential.getKey().slice(0, 6)}${'•'.repeat(24)}`}
                                      </Text>
                                      <IconOnlyButton
                                        renderIcon={isVisible ? ViewOff : View}
                                        iconDescription={`${isVisible ? 'Hide' : 'Show'} ${credential.getName()} API key`}
                                        aria-expanded={isVisible}
                                        aria-controls={`project-api-key-${row.id}`}
                                        kind="ghost"
                                        size="sm"
                                        className="!min-h-0 !shrink-0 !p-1"
                                        onClick={() =>
                                          setVisibleCredentialIds(current => {
                                            const next = new Set(current);
                                            if (next.has(row.id))
                                              next.delete(row.id);
                                            else next.add(row.id);
                                            return next;
                                          })
                                        }
                                      />
                                      <IconOnlyButton
                                        renderIcon={
                                          copiedCredentialId === row.id
                                            ? Checkmark
                                            : Copy
                                        }
                                        iconDescription={`Copy ${credential.getName()} API key`}
                                        kind="ghost"
                                        size="sm"
                                        className="!min-h-0 !shrink-0 !p-1"
                                        onClick={async () => {
                                          try {
                                            await navigator.clipboard.writeText(
                                              credential.getKey(),
                                            );
                                            setCopiedCredentialId(row.id);
                                            window.setTimeout(
                                              () =>
                                                setCopiedCredentialId(
                                                  current =>
                                                    current === row.id
                                                      ? null
                                                      : current,
                                                ),
                                              2000,
                                            );
                                            toast.success('API key copied.');
                                          } catch {
                                            toast.error(
                                              'API key could not be copied. Copy it manually after showing it.',
                                            );
                                          }
                                        }}
                                      />
                                    </div>
                                  ) : cell.info.header === 'role' ? (
                                    <Tag size="sm" type="cool-gray">
                                      {credential.getRolename()}
                                    </Tag>
                                  ) : cell.info.header === 'actions' ? (
                                    <div className="flex items-center gap-0">
                                      <IconOnlyButton
                                        size="md"
                                        kind="ghost"
                                        renderIcon={Unlink}
                                        iconDescription={`Revoke ${credential.getName()}`}
                                        tooltipPosition="bottom"
                                        onClick={() =>
                                          setCredentialPendingRevoke(credential)
                                        }
                                      />
                                    </div>
                                  ) : (
                                    cell.value
                                  )}
                                </TableCell>
                              ))}
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  )}
                </DataTable>
              </ScrollableTableSection>
              <Pagination
                sticky
                page={page}
                pageSize={pageSize}
                pageSizes={[10, 25, 50, 100]}
                totalItems={totalItems}
                onChange={value => {
                  setPage(value.page);
                  setPageSize(value.pageSize);
                }}
              />
            </>
          )}
        </TableSection>
      )}

      {currentProject?.id && (
        <CreateProjectToken
          open={isCreateOpen}
          projectId={currentProject.id}
          auth={{ organizationId, token, userId }}
          onClose={() => setIsCreateOpen(false)}
          onCreated={() => {
            setPage(1);
            setRefresh(value => value + 1);
          }}
        />
      )}
      {credentialPendingRevoke && currentProject?.id && (
        <ConfirmDeleteDialog
          showing
          title="Revoke API key"
          content={`Applications using "${credentialPendingRevoke.getName()}" will lose access immediately.`}
          confirmText="Revoke API key"
          objectName={credentialPendingRevoke.getName()}
          onClose={() => setCredentialPendingRevoke(null)}
          onCancel={() => setCredentialPendingRevoke(null)}
          onConfirm={() => {
            const revokedCredential = credentialPendingRevoke;
            revokeProjectCredential({
              projectId: currentProject.id,
              credentialId: revokedCredential.getId(),
              auth: { organizationId, token, userId },
              callback: (revokeError, response) => {
                if (response?.getSuccess()) {
                  setCredentialPendingRevoke(null);
                  toast.success('API key was revoked.');
                  if (credentials.length === 1 && page > 1) {
                    setPage(value => value - 1);
                  } else {
                    setRefresh(value => value + 1);
                  }
                  return;
                }
                if (!revokeError) {
                  toast.error(
                    response?.getError()?.getHumanmessage() ||
                      'API key could not be revoked. Try again.',
                  );
                  return;
                }
                listProjectCredentials({
                  projectId: currentProject.id,
                  page,
                  pageSize,
                  criteria,
                  auth: { organizationId, token, userId },
                  callback: (_, listResponse) => {
                    if (listResponse?.getSuccess()) {
                      const nextCredentials = listResponse.getDataList();
                      setCredentials(nextCredentials);
                      setTotalItems(
                        listResponse.getPaginated()?.getTotalitem() || 0,
                      );
                      if (
                        !nextCredentials.some(
                          current =>
                            current.getId() === revokedCredential.getId(),
                        )
                      ) {
                        setCredentialPendingRevoke(null);
                        toast.success('API key was revoked.');
                        return;
                      }
                    }
                    toast.error(
                      'API key revocation could not be confirmed. Try again.',
                    );
                  },
                });
              },
            });
          }}
        />
      )}
    </div>
  );
}

export function PersonalCredentialPage() {
  /**
   * authentication
   */
  const { token, authId, projectId } = useCurrentCredential();

  const [isVisible, setIsVisible] = useState(false);
  const [isCopied, setIsCopied] = useState(false);

  const copyToClipboard = async text => {
    try {
      await navigator.clipboard.writeText(text);
      setIsCopied(true);
      setTimeout(() => setIsCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const maskCredential = credential => {
    return credential.substring(0, 6) + '•'.repeat(36);
  };

  /**
   *
   */
  return (
    <div
      data-page-scroll
      className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden"
    >
      <Helmet title="Providers and Models" />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Integrations
          </Text>
        }
        title="Personal Tokens"
        description="Use your personal access token to authenticate platform service requests across projects."
      />
      <DocNoticeBlock docPath="/integrations/rapida-credentials">
        These are your personal access tokens. They are used to authenticate and
        interact with the platform service across all your projects.
      </DocNoticeBlock>
      <section className="grid content-start grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 grow shrink-0 m-4">
        <BaseCard>
          {/* Card header */}
          <div className="px-4 pt-4 pb-3">
            <Text variant="heading-compact-01" className="text-foreground">
              Personal Token
            </Text>
          </div>

          {/* Fields */}
          <div className="px-4 pb-4 border-t border-border-subtle pt-3 space-y-4">
            <FieldSet>
              <FormLabel>Authorization</FormLabel>
              <div className="flex items-center gap-2">
                <Text
                  as="code"
                  variant="code-01"
                  className="min-w-0 flex-1 truncate bg-layer px-3 py-2"
                >
                  {isVisible ? token : maskCredential(token)}
                </Text>
                <div className="flex shrink-0 border border-border-subtle divide-x divide-border-subtle ">
                  <GhostButton
                    size="md"
                    onClick={() => setIsVisible(!isVisible)}
                    title={isVisible ? 'Hide' : 'Show'}
                  >
                    {isVisible ? <ViewOff size={16} /> : <View size={16} />}
                  </GhostButton>
                  <GhostButton
                    size="md"
                    onClick={() => copyToClipboard(token)}
                    title="Copy"
                  >
                    {isCopied ? <Checkmark size={16} /> : <Copy size={16} />}
                  </GhostButton>
                </div>
              </div>
            </FieldSet>
            <FieldSet>
              <FormLabel>x-auth-id</FormLabel>
              <div className="flex items-center gap-2">
                <Text
                  as="code"
                  variant="code-01"
                  className="min-w-0 flex-1 truncate bg-layer px-3 py-2"
                >
                  {isVisible ? authId : maskCredential(authId)}
                </Text>
                <div className="flex shrink-0 border border-border-subtle divide-x divide-border-subtle ">
                  <CopyButton className="h-8 w-8">{authId}</CopyButton>
                </div>
              </div>
            </FieldSet>
            <FieldSet>
              <FormLabel>Project ID</FormLabel>
              <div className="flex items-center gap-2">
                <Text
                  as="code"
                  variant="code-01"
                  className="min-w-0 flex-1 truncate bg-layer px-3 py-2"
                >
                  {isVisible ? projectId : maskCredential(projectId)}
                </Text>
                <div className="flex shrink-0 border border-border-subtle divide-x divide-border-subtle ">
                  <CopyButton className="h-8 w-8">{projectId}</CopyButton>
                </div>
              </div>
            </FieldSet>
          </div>
        </BaseCard>
      </section>
    </div>
  );
}
