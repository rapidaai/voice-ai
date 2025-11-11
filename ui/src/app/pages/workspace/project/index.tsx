import { useCallback, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/Helmet';
import {
  ArchiveProjectResponse,
  GetAllProjectResponse,
  Project,
} from '@rapidaai/react';
import { CreateProjectDialog } from '@/app/components/base/modal/create-project-modal';
import { GetAllProject, DeleteProject } from '@rapidaai/react';
import { useCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { useRapidaStore } from '@/hooks';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { ServiceError } from '@rapidaai/react';
import { IBlueButton, IButton } from '@/app/components/Form/Button';
import { Plus, RotateCw } from 'lucide-react';
import { Table } from '@/app/components/base/tables/table';
import { TableHead } from '@/app/components/base/tables/table-head';
import { TableBody } from '@/app/components/base/tables/table-body';
import { TableRow } from '@/app/components/base/tables/table-row';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { ProjectUserGroupAvatar } from '@/app/components/Avatar/ProjectUserGroupAvatar';
import { toHumanReadableDate } from '@/styles/media';
import { RoleIndicator } from '@/app/components/indicators/role';
import { ProjectOption } from '@/app/pages/workspace/project/project-options';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { TableSection } from '@/app/components/sections/table-section';
import { connectionConfig } from '@/configs';
export function ProjectPage() {
  /**
   *
   */
  const [createProjectModalOpen, setCreateProjectModalOpen] = useState(false);

  /**
   * loading context
   */
  const { showLoader, hideLoader } = useRapidaStore();

  /**
   * Credentials
   */
  const [userId, token] = useCredential();

  /**
   * List of projects
   */
  const [projects, setProjects] = useState<Project[]>([]);

  /**
   * pagination and search capabilities
   */

  const [page, setPage] = useState<number>(1);
  const [pageSize, setPageSize] = useState(10);
  const [totalCount, setTotalCount] = useState(0);

  /**
   * filter apply
   */
  const [criteria, _] = useState<{ key: string; value: string }[]>([]);

  /**
   *
   */
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
      }
    },
    [],
  );

  /**
   *
   * @param page
   * @param pageSize
   * @param criteria
   * @returns
   */
  const getAllProject = (
    page: number,
    pageSize: number,
    criteria: { key: string; value: string }[],
  ) => {
    showLoader();
    return GetAllProject(
      connectionConfig,
      page,
      pageSize,
      criteria,
      afterGettingProject,
      {
        authorization: token,
        'x-auth-id': userId,
      },
    );
  };

  /**
   *
   */
  useEffect(() => {
    getAllProject(page, pageSize, criteria);
  }, [page, pageSize, criteria]);

  /**
   *
   * @param projectId
   */
  const onDeleteProject = (projectId: string) => {
    DeleteProject(
      connectionConfig,
      projectId,
      (err: ServiceError | null, apr: ArchiveProjectResponse | null) => {
        if (err) {
          return;
        }
        if (apr?.getSuccess()) {
          const newList = projects?.filter(p => p.getId() !== apr.getId());
          setProjects(newList);
        }
      },
      {
        authorization: token,
        'x-auth-id': userId,
      },
    );
  };

  return (
    <>
      <Helmet title="Projects" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Projects</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${projects.length}/${totalCount}`}
          </div>
        </div>
        <div className="flex divide-x dark:divide-gray-800">
          <IButton
            onClick={() => {
              getAllProject(page, pageSize, criteria);
            }}
          >
            Reload Project
            <RotateCw className="w-4 h-4 ml-1.5" />
          </IButton>
          <IBlueButton
            onClick={() => {
              setCreateProjectModalOpen(true);
            }}
          >
            Create new project
            <Plus className="w-4 h-4 ml-1.5" />
          </IBlueButton>
        </div>
      </PageHeaderBlock>
      <BluredWrapper className="p-0">
        <SearchIconInput className="bg-light-background" />
        <TablePagination
          currentPage={page}
          onChangeCurrentPage={setPage}
          totalItem={totalCount}
          pageSize={pageSize}
          onChangePageSize={setPageSize}
        />
      </BluredWrapper>
      <TableSection>
        <Table className="bg-white dark:bg-gray-900">
          <TableHead
            isActionable
            columns={[
              { name: 'Name', key: 'name' },
              { name: 'Date Created', key: 'createdDate' },
              { name: 'Your Role', key: 'role' },
              { name: 'Collaborators', key: 'collaborators' },
            ]}
          ></TableHead>
          <TableBody className="">
            {projects.map(project => (
              <TableRow key={project.getId()}>
                <TableCell>{project.getName()}</TableCell>
                <TableCell>
                  {project.getCreateddate() &&
                    toHumanReadableDate(project.getCreateddate()!)}
                </TableCell>
                <TableCell>
                  <RoleIndicator role={'SUPER_ADMIN'} />
                </TableCell>
                <TableCell>
                  <ProjectUserGroupAvatar
                    members={project
                      .getMembersList()
                      .map(m => ({ name: m.getName() }))}
                    size={7}
                    projectId={project.getId()}
                  />
                </TableCell>
                <TableCell>
                  <ProjectOption
                    project={project.toObject()}
                    afterUpdateProject={() => {
                      getAllProject(page, pageSize, criteria);
                    }}
                    onDelete={() => onDeleteProject(project.getId())}
                  ></ProjectOption>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <CreateProjectDialog
          modalOpen={createProjectModalOpen}
          setModalOpen={setCreateProjectModalOpen}
          afterCreateProject={() => {
            getAllProject(page, pageSize, criteria);
          }}
        />
      </TableSection>
    </>
  );
}
