import { CustomLink, CustomLinkProps } from '@/app/components/custom-link';
import { MultiplePills } from '@/app/components/Pill';
import { Tooltip } from '@/app/components/Tooltip';
import { cn } from '@/styles/media';
import { FC, HTMLAttributes } from 'react';

interface CardProps extends HTMLAttributes<HTMLDivElement> {}
export const Card: FC<CardProps> = ({ children, className, ...props }) => {
  return (
    <div
      className={cn(
        'dark:bg-gray-950 bg-white relative flex flex-col overflow-hidden p-4 h-fit',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
};

interface ClickableCardProps extends CardProps {}

export const ClickableCard: FC<ClickableCardProps & CustomLinkProps> = ({
  to,
  isExternal,
  children,
  className,
}) => {
  return (
    <CustomLink to={to} isExternal={isExternal}>
      <Card className={cn('group hover:shadow-md', className)}>{children}</Card>
    </CustomLink>
  );
};

interface CardTitleProps extends HTMLAttributes<HTMLDivElement> {
  status?: string;
  title?: string;
  children?: any;
}
export const CardTitle: FC<CardTitleProps> = ({
  title,
  status,
  children,
  className,
}) => {
  return (
    <div className={cn('capitalize', className)}>
      <span className="">
        {title}
        {children}
      </span>
      {status === 'active' && (
        <Tooltip
          icon={
            <span className="relative flex h-2 w-2 ml-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-[2px] bg-blue-400 opacity-75"></span>
              <span className="relative inline-flex rounded-[2px] h-2 w-2 bg-blue-500"></span>
            </span>
          }
        >
          <p>Active and available to use right now</p>
        </Tooltip>
      )}
    </div>
  );
};
interface CardDescriptionProps extends HTMLAttributes<HTMLDivElement> {
  description?: string;
  children?: any;
}
export const CardDescription: FC<CardDescriptionProps> = ({
  description,
  className,
  children,
}) => {
  return (
    <p
      className={cn(
        'mt-1 opacity-70 text-sm leading-normal line-clamp-2',
        className,
      )}
    >
      {description}
      {children}
    </p>
  );
};

interface CardTagProps extends HTMLAttributes<HTMLDivElement> {
  tags?: string[];
}
export const CardTag: FC<CardTagProps> = ({ tags, className }) => {
  return (
    <MultiplePills
      tags={tags}
      className={cn('rounded-[2px] w-fit px-4 text-sm', className)}
    />
  );
};
