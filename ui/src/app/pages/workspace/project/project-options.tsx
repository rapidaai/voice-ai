import { useState } from 'react';
import { UpdateProjectDialog } from '@/app/components/base/modal/update-project-modal';
import { Project } from '@rapidaai/react';
import { CardOptionMenu } from '@/app/components/Menu';

export const ProjectOption = (props: {
  project: Project.AsObject;
  afterUpdateProject: () => void;
  onDelete: () => void;
}) => {
  const [projectUpdateModalOpen, setProjectUpdateModalOpen] =
    useState<boolean>(false);

  return (
    <>
      <UpdateProjectDialog
        existingProject={props.project}
        modalOpen={projectUpdateModalOpen}
        setModalOpen={setProjectUpdateModalOpen}
        afterUpdateProject={props.afterUpdateProject}
      ></UpdateProjectDialog>
      <CardOptionMenu
        options={[
          {
            option: 'Update project details',
            onActionClick: () => {
              setProjectUpdateModalOpen(!projectUpdateModalOpen);
            },
          },
        ]}
      />
    </>
  );
};
