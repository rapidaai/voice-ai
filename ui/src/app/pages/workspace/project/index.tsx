import { useCallback, useContext, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import {
  ArchiveProjectResponse,
  GetAllProjectResponse,
  Project,
  ServiceError,
} from '@rapidaai/react';
import { CreateProjectDialog } from '@/app/components/dialogs/workspace';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { Pagination, Text } from '@/app/components/ui/primitives';
import { Add, Edit, Folders, Renew, TrashCan } from '@carbon/icons-react';
import {
  DataTable,
  DataTableSkeleton,
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableCell,
  TableToolbar,
  TableBatchAction,
  TableBatchActions,
  TableToolbarContent,
  TableToolbarSearch,
  Button,
  TableSelectRow,
} from '@carbon/react';
import { ProjectUserGroupAvatar } from '@/app/components/domain/avatar/project-user-group-avatar';
import { toHumanReadableDate } from '@/utils/date';
import { RoleIndicator } from '@/app/components/domain/indicators/role';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { ConfirmDeleteDialog } from '@/app/components/dialogs/shared';
import { AuthContext } from '@/context/auth-context';
import { UpdateProjectDialog } from '@/app/components/dialogs/workspace';
import { CarbonIconIndicator } from '@/app/components/ui/feedback';
import { deleteWorkspaceProject, listWorkspaceProjects } from '@/clients';
import { EmptyState } from '@/app/components/ui/feedback';

const headers = [
  { key: 'name', header: 'Name' },
  { key: 'createdDate', header: 'Date Created' },
  { key: 'role', header: 'Your Role' },
  { key: 'collaborators', header: 'Collaborators' },
  { key: 'status', header: 'Status' },
];

const skeletonHeaders = [{ key: 'select', header: '' }, ...headers];

export function ProjectPage() {
  const [createProjectModalOpen, setCreateProjectModalOpen] = useState(false);
  const [userId, token] = useCredential();
  const [projects, setProjects] = useState<Project[]>([]);
  const [isFetching, setIsFetching] = useState(true);
  const [page, setPage] = useState<number>(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalCount, setTotalCount] = useState(0);
  const [criteria] = useState<{ key: string; value: string }[]>([]);
  const [projectPendingDelete, setProjectPendingDelete] =
    useState<Project | null>(null);
  const [projectPendingUpdate, setProjectPendingUpdate] =
    useState<Project | null>(null);
  const { projectRoles } = useContext(AuthContext);

  const afterGettingProject = useCallback(
    (err: ServiceError | null, alpr: GetAllProjectResponse | null) => {
      setIsFetching(false);
      if (err) {
        toast.error('Unable to process your request. please try again later.');
        return;
      }
      if (alpr?.getSuccess()) {
        setProjects(alpr.getDataList());
        let paginated = alpr.getPaginated();
        if (paginated) {
          setTotalCount(paginated.getTotalitem());
        }
        return;
      }
      toast.error(
        alpr?.getError()?.getHumanmessage() ||
          'Unable to process your request. please try again later.',
      );
    },
    [],
  );

  const getAllProject = useCallback(
    (
      page: number,
      pageSize: number,
      criteria: { key: string; value: string }[],
    ) => {
      setIsFetching(true);
      return listWorkspaceProjects({
        page,
        pageSize,
        criteria,
        auth: { token, userId },
        callback: afterGettingProject,
      });
    },
    [afterGettingProject, token, userId],
  );

  useEffect(() => {
    getAllProject(page, pageSize, criteria);
  }, [getAllProject, page, pageSize, criteria]);

  const onDeleteProject = (projectId: string) => {
    deleteWorkspaceProject({
      projectId,
      auth: { token, userId },
      callback: (
        err: ServiceError | null,
        apr: ArchiveProjectResponse | null,
      ) => {
        if (err) {
          setProjectPendingDelete(null);
          return;
        }
        if (apr?.getSuccess()) {
          const newList = projects?.filter(p => p.getId() !== apr.getId());
          setProjects(newList);
          setProjectPendingDelete(null);
        }
      },
    });
  };

  const tableRows = projects.map(project => ({
    id: project.getId(),
    name: project.getName(),
    createdDate: project.getCreateddate()
      ? toHumanReadableDate(project.getCreateddate()!)
      : '',
    role:
      projectRoles?.find(role => role.projectid === project.getId())?.role ||
      '',
    collaborators: project
      .getMembersList()
      .map(member => member.getName())
      .join(', '),
    status: project.getStatus?.() || '',
  }));

  return (
    <div
      data-page-scroll
      className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden bg-surface"
    >
      <Helmet title="Projects" />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Organization
          </Text>
        }
        title="Projects"
        description="Organize assistants, endpoints, credentials, and collaborators across your workspace."
        contextualActions={
          <Text variant="label-01" className="text-muted tabular-nums">
            {projects.length} of {totalCount}
          </Text>
        }
        pageActions={
          <Button
            className="w-full md:w-auto"
            size="md"
            renderIcon={Add}
            onClick={() => setCreateProjectModalOpen(true)}
          >
            Create new project
          </Button>
        }
      />
      {isFetching ? (
        <TableSection className="min-h-0 overflow-hidden bg-surface">
          <DataTableSkeleton
            headers={skeletonHeaders}
            columnCount={skeletonHeaders.length}
            rowCount={5}
            showHeader={false}
            showToolbar
            size="lg"
            className="w-full"
          />
        </TableSection>
      ) : (
        <TableSection className="min-h-0 bg-surface">
          <DataTable rows={tableRows} headers={headers} radio>
            {({
              rows,
              headers: tableHeaders,
              selectedRows,
              getBatchActionProps,
              getHeaderProps,
              getRowProps,
              getSelectionProps,
              getTableProps,
              getToolbarProps,
              onInputChange,
            }) => {
              const batchActionProps = getBatchActionProps();
              const selectedProject = selectedRows[0]
                ? projects.find(
                    project => project.getId() === selectedRows[0].id,
                  )
                : undefined;

              return (
                <>
                  <PageHeaderStickyToolbar>
                    <TableToolbar {...getToolbarProps()}>
                      {selectedProject && (
                        <TableBatchActions
                          {...batchActionProps}
                          shouldShowBatchActions
                        >
                          <TableBatchAction
                            renderIcon={Edit}
                            iconDescription="Edit"
                            onClick={() => {
                              setProjectPendingUpdate(selectedProject);
                              batchActionProps.onCancel();
                            }}
                          >
                            Edit
                          </TableBatchAction>
                          <TableBatchAction
                            renderIcon={TrashCan}
                            iconDescription="Delete"
                            onClick={() => {
                              setProjectPendingDelete(selectedProject);
                              batchActionProps.onCancel();
                            }}
                          >
                            Delete
                          </TableBatchAction>
                        </TableBatchActions>
                      )}
                      <TableToolbarContent>
                        <TableToolbarSearch
                          placeholder="Search projects..."
                          onChange={onInputChange}
                        />
                        <Button
                          hasIconOnly
                          renderIcon={Renew}
                          iconDescription="Refresh"
                          kind="ghost"
                          onClick={() =>
                            getAllProject(page, pageSize, criteria)
                          }
                          tooltipPosition="bottom"
                        />
                      </TableToolbarContent>
                    </TableToolbar>
                  </PageHeaderStickyToolbar>

                  {projects.length > 0 ? (
                    <>
                      <div className="shrink-0 overflow-x-auto">
                        <Table {...getTableProps()} className="min-w-max">
                          <TableHead>
                            <TableRow>
                              <TableHeader className="!w-12" />
                              {tableHeaders.map(header => (
                                <TableHeader
                                  {...getHeaderProps({ header })}
                                  key={header.key}
                                >
                                  {header.header}
                                </TableHeader>
                              ))}
                            </TableRow>
                          </TableHead>
                          <TableBody>
                            {rows.map(tableRow => {
                              const project = projects.find(
                                candidate => candidate.getId() === tableRow.id,
                              );
                              if (!project) return null;

                              return (
                                <TableRow
                                  {...getRowProps({ row: tableRow })}
                                  key={tableRow.id}
                                >
                                  <TableSelectRow
                                    {...getSelectionProps({ row: tableRow })}
                                  />
                                  <TableCell>{project.getName()}</TableCell>
                                  <TableCell>
                                    {project.getCreateddate() &&
                                      toHumanReadableDate(
                                        project.getCreateddate()!,
                                      )}
                                  </TableCell>
                                  <TableCell>
                                    <RoleIndicator
                                      role={
                                        projectRoles?.find(
                                          role =>
                                            role.projectid === project.getId(),
                                        )?.role
                                      }
                                    />
                                  </TableCell>
                                  <TableCell>
                                    <ProjectUserGroupAvatar
                                      members={project
                                        .getMembersList()
                                        .map(member => ({
                                          name: member.getName(),
                                        }))}
                                      size="sm"
                                    />
                                  </TableCell>
                                  <TableCell>
                                    <CarbonIconIndicator
                                      state={project.getStatus?.()}
                                    />
                                  </TableCell>
                                </TableRow>
                              );
                            })}
                          </TableBody>
                        </Table>
                      </div>
                      <Pagination
                        sticky
                        totalItems={totalCount}
                        page={page}
                        pageSize={pageSize}
                        pageSizes={[10, 20, 50]}
                        onChange={({ page: newPage, pageSize: newSize }) => {
                          setPage(newPage);
                          setPageSize(newSize);
                        }}
                      />
                    </>
                  ) : (
                    <div className="flex flex-1 items-center justify-center">
                      <EmptyState
                        icon={Folders}
                        title="No projects"
                        subtitle="Create a project to organize assistants, endpoints, credentials, and collaborators."
                        action="Create new project"
                        actionIcon={Add}
                        onAction={() => setCreateProjectModalOpen(true)}
                      />
                    </div>
                  )}
                </>
              );
            }}
          </DataTable>
        </TableSection>
      )}
      <CreateProjectDialog
        modalOpen={createProjectModalOpen}
        setModalOpen={setCreateProjectModalOpen}
        afterCreateProject={() => {
          getAllProject(page, pageSize, criteria);
        }}
      />
      {projectPendingUpdate && (
        <UpdateProjectDialog
          existingProject={projectPendingUpdate.toObject()}
          modalOpen={Boolean(projectPendingUpdate)}
          setModalOpen={open => {
            if (!open) {
              setProjectPendingUpdate(null);
            }
          }}
          afterUpdateProject={() => {
            setProjectPendingUpdate(null);
            getAllProject(page, pageSize, criteria);
          }}
        />
      )}
      <ConfirmDeleteDialog
        showing={Boolean(projectPendingDelete)}
        title="Delete project"
        content={
          projectPendingDelete
            ? `This will delete "${projectPendingDelete.getName()}". Type the project name to confirm.`
            : ''
        }
        objectName={projectPendingDelete?.getName() || ''}
        confirmText="Delete project"
        onConfirm={() => {
          if (projectPendingDelete) {
            onDeleteProject(projectPendingDelete.getId());
          }
        }}
        onCancel={() => {
          setProjectPendingDelete(null);
        }}
        onClose={() => {
          setProjectPendingDelete(null);
        }}
      />
    </div>
  );
}
