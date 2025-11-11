import { cn } from '@/styles/media';
import { FC, HTMLAttributes } from 'react';
import { NavLink } from 'react-router-dom';

interface TabProps extends HTMLAttributes<HTMLDivElement> {
  isActive?: boolean;
}

interface LinkTabProps extends TabProps {
  to: string;
}

export const Tab: FC<TabProps> = ({ isActive, children, ...props }) => {
  return (
    <div
      {...props}
      className={cn(
        'group px-2 border-b-[3px] border-transparent -mb-[0.2rem] cursor-pointer',
        isActive ? 'border-blue-500! text-blue-500' : '',
      )}
    >
      <div className="capitalize px-3 py-3 group-hover:bg-blue-600/5 dark:group-hover:bg-blue-950/50">
        {children}
      </div>
    </div>
  );
};

export const TabLink: FC<LinkTabProps> = ({ to, children }) => {
  return (
    <NavLink
      to={to}
      className={({ isActive }) => {
        return cn(
          'group cursor-pointer hover:bg-gray-500/10',
          isActive ? 'bg-gray-500/10 text-blue-500' : '',
        );
      }}
    >
      <div className="px-6 py-2 font-medium whitespace-nowrap tracking-wide text-pretty">
        {children}
      </div>
    </NavLink>
  );
};
export const SideTabLink: FC<LinkTabProps> = props => {
  return (
    <NavLink
      to={props.to}
      className={({ isActive }) =>
        cn(
          'flex items-center px-5 py-2 relative hover:text-blue-600',
          isActive && 'dark:bg-gray-950 bg-gray-200 text-blue-600',
          props.className,
        )
      }
    >
      {props.children}
    </NavLink>
  );
};

export const SideTab: FC<LinkTabProps> = props => {
  return (
    <div
      onClick={props.onClick}
      className={cn(
        'flex items-center px-5 py-2 relative hover:text-blue-600 cursor-pointer',
        props.className,
        props.isActive === true &&
          'dark:bg-gray-950 bg-gray-200 border-r-[3px] border-blue-600 text-blue-600',
      )}
    >
      {props.children}
    </div>
  );
};
