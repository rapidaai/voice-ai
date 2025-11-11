import React, { useState } from 'react';
import { InviteUserDialog } from '@/app/components/base/modal/invite-user-modal';
import { TextImage } from '../Image/TextImage';

export function ProjectUserGroupAvatar(props: {
  size?: 7 | 8 | 9;
  members: { name: string }[];
  projectId: string;
}) {
  const [createUserModalOpen, setCreateUserModalOpen] = useState(false);
  return (
    <ul className="flex flex-wrap justify-center sm:justify-start mb-8 sm:mb-0 -space-x-2 -ml-px">
      {props.members.map((usr, idx) => {
        return (
          <li key={idx}>
            <div className="block rounded-[2px] border border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600">
              <TextImage size={props.size} name={usr.name}></TextImage>
            </div>
          </li>
        );
      })}

      <li>
        <div
          onClick={() => {
            setCreateUserModalOpen(true);
          }}
          className="-ml-px block rounded-[2px] border border-gray-200 dark:border-gray-700 hover:border-gray-300 dark:hover:border-gray-600"
        >
          <TextImage size={props.size} name={'+'}></TextImage>
        </div>
        <InviteUserDialog
          modalOpen={createUserModalOpen}
          projectId={props.projectId}
          setModalOpen={setCreateUserModalOpen}
        />
      </li>
    </ul>
  );
}
