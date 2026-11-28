import type {
  ComponentType,
  ElementType,
  KeyboardEvent,
  MouseEvent,
  MouseEventHandler,
  ReactNode,
} from 'react';
import {
  OverflowMenu as CarbonOverflowMenu,
  MenuItem as CarbonMenuItem,
  MenuItemDivider,
} from '@carbon/react';
import { cn } from '@/utils';

type OverflowMenuSize = 'xs' | 'sm' | 'md' | 'lg';
type MenuDirection = 'top' | 'bottom';
type MenuAlignment = 'top-start' | 'top-end' | 'bottom-start' | 'bottom-end';

interface CarbonV12OverflowMenuProps {
  children: ReactNode;
  className?: string;
  label: string;
  menuAlignment: MenuAlignment;
  renderIcon?: ElementType;
  size: OverflowMenuSize;
}

const CarbonV12OverflowMenu =
  CarbonOverflowMenu as ComponentType<CarbonV12OverflowMenuProps>;

export interface OverflowMenuProps {
  children: ReactNode;
  /** Styles the v12 menu container. Target descendant `.cds--overflow-menu` for trigger styles. */
  className?: string;
  size: OverflowMenuSize;
  iconDescription?: string;
  renderIcon?: ElementType;
  direction?: MenuDirection;
  flipped?: boolean;
  'aria-label'?: string;
}

export interface OverflowMenuItemProps {
  className?: string;
  itemText: string;
  disabled?: boolean;
  isDelete?: boolean;
  hasDivider?: boolean;
  href?: string;
  onClick?: MouseEventHandler;
  dangerDescription?: string;
  renderIcon?: ElementType;
}

/**
 * Carbon OverflowMenu: a vertical dot menu that reveals a list of actions.
 */
export function OverflowMenu({
  children,
  className,
  size,
  iconDescription = 'More actions',
  renderIcon,
  direction = 'bottom',
  flipped = false,
  'aria-label': ariaLabel,
}: OverflowMenuProps) {
  const menuAlignment: MenuAlignment = `${direction}-${
    flipped ? 'end' : 'start'
  }`;

  return (
    <CarbonV12OverflowMenu
      className={cn(className)}
      size={size}
      label={ariaLabel ?? iconDescription}
      renderIcon={renderIcon}
      menuAlignment={menuAlignment}
    >
      {children}
    </CarbonV12OverflowMenu>
  );
}

/**
 * Carbon OverflowMenuItem: a single action inside an OverflowMenu.
 * Use `isDelete` for danger/destructive items, `hasDivider` for visual separation.
 */
export function OverflowMenuItem({
  className,
  itemText,
  disabled = false,
  isDelete = false,
  hasDivider = false,
  href,
  onClick,
  dangerDescription,
  renderIcon,
}: OverflowMenuItemProps) {
  const handleClick = (
    event: KeyboardEvent<HTMLLIElement> | MouseEvent<HTMLLIElement>,
  ) => {
    onClick?.(event as unknown as MouseEvent<HTMLElement>);
    if (href && !event.defaultPrevented) {
      window.location.assign(href);
    }
  };

  return (
    <>
      {hasDivider && <MenuItemDivider />}
      <CarbonMenuItem
        className={cn(className)}
        label={itemText}
        disabled={disabled}
        kind={isDelete ? 'danger' : 'default'}
        onClick={handleClick}
        dangerDescription={dangerDescription}
        renderIcon={renderIcon}
      />
    </>
  );
}
