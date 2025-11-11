import { FC, HTMLAttributes } from 'react';

export const ModalFormBlock: FC<HTMLAttributes<HTMLFormElement>> = props => {
  return (
    <form
      className="w-[750px] max-w-full bg-light-background dark:bg-gray-900 relative items-start"
      {...props}
    >
      {props.children}
    </form>
  );
};
