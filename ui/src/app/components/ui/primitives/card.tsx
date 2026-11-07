import {
  CustomLink,
  CustomLinkProps,
} from '@/app/components/ui/primitives/custom-link';
import { MultiplePills } from '@/app/components/ui/primitives/pill';
import { Tooltip } from '@/app/components/ui/primitives/tooltip';
import { cn } from '@/utils';
import type {
  HTMLAttributes,
  KeyboardEvent,
  MouseEvent,
  ReactNode,
} from 'react';
import { Link } from 'react-router-dom';
import { CornerBorderOverlay } from '@/app/components/ui/primitives/corner-border';
import { Tile as CarbonTile } from '@carbon/react';

// Shared card primitives.

/** Static card: background + corner-bracket hover, no interactivity. */
export function BaseCard({
  className,
  children,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <CarbonTile
      className={cn(
        'bg-layer border border-border-subtle relative group flex flex-col transition-colors duration-100',
        className,
      )}
      {...props}
    >
      <CornerBorderOverlay />
      {children}
    </CarbonTile>
  );
}

/** Navigable card: wraps content in a react-router Link. */
export function LinkCard({
  to,
  className,
  children,
}: {
  to: string;
  className?: string;
  children?: ReactNode;
}) {
  return (
    <Link
      to={to}
      className={cn(
        'bg-surface border border-border-subtle relative group flex flex-col transition-colors duration-100',
        className,
      )}
    >
      <CornerBorderOverlay />
      {children}
    </Link>
  );
}

/** Actionable card: div with role="button", keyboard support, and focus ring. */
export function ActionCard({
  className,
  children,
  onClick,
  onKeyDown,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' || e.key === ' ')
      onClick?.(e as unknown as MouseEvent<HTMLDivElement>);
    onKeyDown?.(e);
  };

  return (
    <CarbonTile
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={handleKeyDown}
      className={cn(
        'bg-layer border border-border-subtle relative group flex flex-col transition-colors duration-100 cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary',
        className,
      )}
      {...props}
    >
      <CornerBorderOverlay />
      {children}
    </CarbonTile>
  );
}

// Existing card primitives kept for compatibility.

interface CardProps extends HTMLAttributes<HTMLDivElement> {}
export function Card({ children, className, ...props }: CardProps) {
  return (
    <CarbonTile
      className={cn(
        // Carbon tile: no border radius, 1px border
        'bg-layer relative group flex flex-col overflow-hidden p-4 h-fit border rounded-none',
        className,
      )}
      {...props}
    >
      <CornerBorderOverlay />
      {children}
    </CarbonTile>
  );
}

interface ClickableCardProps extends CardProps {}

export function ClickableCard({
  to,
  isExternal,
  children,
  className,
}: ClickableCardProps & CustomLinkProps) {
  return (
    <CustomLink to={to} isExternal={isExternal}>
      {/* Carbon clickable tile: no shadow, hover = subtle bg tint */}
      <Card
        className={cn(
          'group cursor-pointer hover:bg-layer transition-colors duration-100',
          className,
        )}
      >
        {children}
      </Card>
    </CustomLink>
  );
}

interface CardTitleProps extends HTMLAttributes<HTMLDivElement> {
  status?: string;
  title?: string;
  children?: ReactNode;
}
export function CardTitle({
  title,
  status,
  children,
  className,
}: CardTitleProps) {
  return (
    <div className={cn('capitalize', className)}>
      <span className="text-sm/6 font-medium">
        {title}
        {children}
      </span>
      {status === 'active' && (
        <Tooltip
          icon={
            <span className="relative flex h-2 w-2 ml-2">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-[2px] bg-support-info opacity-75"></span>
              <span className="relative inline-flex rounded-[2px] h-2 w-2 bg-support-info"></span>
            </span>
          }
        >
          <p>Active and available to use right now</p>
        </Tooltip>
      )}
    </div>
  );
}
interface CardDescriptionProps extends HTMLAttributes<HTMLDivElement> {
  description?: string;
  children?: ReactNode;
}
export function CardDescription({
  description,
  className,
  children,
}: CardDescriptionProps) {
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
}

interface CardTagProps extends HTMLAttributes<HTMLDivElement> {
  tags?: string[];
}
export function CardTag({ tags, className }: CardTagProps) {
  return (
    <MultiplePills
      tags={tags}
      className={cn('rounded-[2px] w-fit px-4 text-sm', className)}
    />
  );
}
