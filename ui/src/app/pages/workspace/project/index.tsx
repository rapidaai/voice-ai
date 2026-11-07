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
import { useRapidaStore } from '@/stores/app';
import { PrimaryButton } from '@/app/components/ui/primitives';
import { Pagination } from '@/app/components/ui/primitives';
import { Add, Edit, Folders, Renew, TrashCan } from '@carbon/icons-react';
import {
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
  RadioButton,
} from '@carbon/react';
import { ProjectUserGroupAvatar } from '@/app/components/domain/avatar/project-user-group-avatar';
import { toHumanReadableDate } from '@/utils/date';
import { RoleIndicator } from '@/app/components/domain/indicators/role';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { PageTitleWithCount } from '@/app/components/layout/blocks/page-title-with-count';
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

export function ProjectPage() {
  const [createProjectModalOpen, setCreateProjectModalOpen] = useState(false);
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [userId, token] = useCredential();
  const [projects, setProjects] = useState<Project[]>([]);
  const [page, setPage] = useState<number>(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalCount, setTotalCount] = useState(0);
  const [criteria] = useState<{ key: string; value: string }[]>([]);
  const [projectPendingDelete, setProjectPendingDelete] =
    useState<Project | null>(null);
  const [projectPendingUpdate, setProjectPendingUpdate] =
    useState<Project | null>(null);
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(
    null,
  );
  const { projectRoles } = useContext(AuthContext);
  const selectedProject =
    projects.find(project => project.getId() === selectedProjectId) || null;

  const afterGettingProject = useCallback(
    (err: ServiceError | null, alpr: GetAllProjectResponse | null) => {
      hideLoader();
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

  const getAllProject = (
    page: number,
    pageSize: number,
    criteria: { key: string; value: string }[],
  ) => {
    showLoader();
    return listWorkspaceProjects({
      page,
      pageSize,
      criteria,
      auth: { token, userId },
      callback: afterGettingProject,
    });
  };

  useEffect(() => {
    getAllProject(page, pageSize, criteria);
  }, [page, pageSize, criteria]);

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
          setSelectedProjectId(null);
        }
      },
    });
  };

  return (
    <>
      <Helmet title="Projects" />
      <PageHeaderBlock>
        <PageTitleWithCount count={projects.length} total={totalCount}>
          Projects
        </PageTitleWithCount>
      </PageHeaderBlock>
      <TableToolbar>
        <TableBatchActions
          shouldShowBatchActions={Boolean(selectedProject)}
          totalSelected={selectedProject ? 1 : 0}
          onCancel={() => setSelectedProjectId(null)}
          totalCount={projects.length}
        >
          <TableBatchAction
            renderIcon={Edit}
            onClick={() => {
              if (selectedProject) {
                setProjectPendingUpdate(selectedProject);
              }
            }}
          >
            Update project details
          </TableBatchAction>
          <TableBatchAction
            className="cds--btn--danger"
            renderIcon={TrashCan}
            onClick={() => {
              if (selectedProject) {
                setProjectPendingDelete(selectedProject);
              }
            }}
          >
            Delete project
          </TableBatchAction>
        </TableBatchActions>
        <TableToolbarContent>
          <TableToolbarSearch placeholder="Search projects..." />
          <Button
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh"
            kind="ghost"
            onClick={() => getAllProject(page, pageSize, criteria)}
            tooltipPosition="bottom"
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            isLoading={loading}
            onClick={() => setCreateProjectModalOpen(true)}
          >
            Create new project
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>
      {!loading && projects.length === 0 ? (
        <EmptyState
          icon={Folders}
          title="No projects"
          subtitle="Create a project to organize assistants, endpoints, credentials, and collaborators."
          action="Create new project"
          actionIcon={Add}
          onAction={() => setCreateProjectModalOpen(true)}
        />
      ) : (
        <TableSection>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader className="!w-12" />
                {headers.map(h => (
                  <TableHeader key={h.key}>{h.header}</TableHeader>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {projects.map(project => (
                <TableRow
                  key={project.getId()}
                  isSelected={selectedProjectId === project.getId()}
                  onClick={() =>
                    setSelectedProjectId(
                      selectedProjectId === project.getId()
                        ? null
                        : project.getId(),
                    )
                  }
                  className="cursor-pointer"
                >
                  <TableCell
                    className="!w-12 !pr-0"
                    onClick={e => e.stopPropagation()}
                  >
                    <RadioButton
                      id={`project-select-${project.getId()}`}
                      name="project-select"
                      labelText=""
                      hideLabel
                      checked={selectedProjectId === project.getId()}
                      onChange={() =>
                        setSelectedProjectId(
                          selectedProjectId === project.getId()
                            ? null
                            : project.getId(),
                        )
                      }
                    />
                  </TableCell>
                  <TableCell>{project.getName()}</TableCell>
                  <TableCell>
                    {project.getCreateddate() &&
                      toHumanReadableDate(project.getCreateddate()!)}
                  </TableCell>
                  <TableCell>
                    <RoleIndicator
                      role={
                        projectRoles?.find(p => p.projectid === project.getId())
                          ?.role
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <ProjectUserGroupAvatar
                      members={project
                        .getMembersList()
                        .map(m => ({ name: m.getName() }))}
                      size="sm"
                    />
                  </TableCell>
                  <TableCell>
                    <CarbonIconIndicator state={project.getStatus?.()} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <Pagination
            totalItems={totalCount}
            page={page}
            pageSize={pageSize}
            pageSizes={[10, 20, 50]}
            onChange={({ page: newPage, pageSize: newSize }) => {
              setPage(newPage);
              setPageSize(newSize);
            }}
          />
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
            setSelectedProjectId(null);
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
          setSelectedProjectId(null);
        }}
        onClose={() => {
          setProjectPendingDelete(null);
          setSelectedProjectId(null);
        }}
      />
    </>
  );
}
