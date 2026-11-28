import { Fragment, type MouseEventHandler, type ReactNode } from 'react';
import {
  Breadcrumb as CarbonBreadcrumb,
  BreadcrumbItem as CarbonBreadcrumbItem,
  BreadcrumbSkeleton,
} from '@carbon/react';
import { cn } from '@/utils';
import { OverflowMenu, OverflowMenuItem } from './overflow-menu';

export interface BreadcrumbItemData {
  label: ReactNode;
  overflowLabel?: string;
  href?: string;
  className?: string;
  isCurrentPage?: boolean;
  onClick?: MouseEventHandler<HTMLElement>;
  /** Render custom content inside the BreadcrumbItem instead of a plain link. */
  render?: () => ReactNode;
}

export interface CarbonBreadcrumbProps {
  items: BreadcrumbItemData[];
  className?: string;
  noTrailingSlash?: boolean;
  isLoading?: boolean;
  /** Maximum visible items before middle ancestors move into overflow. Minimum: 2. */
  maxVisibleItems?: number;
}

/** Carbon Breadcrumb: renders items or a skeleton placeholder when loading. */
export function Breadcrumb({
  items,
  className,
  noTrailingSlash = true,
  isLoading = false,
  maxVisibleItems,
}: CarbonBreadcrumbProps) {
  if (isLoading) {
    return <BreadcrumbSkeleton className={cn(className)} />;
  }

  const shouldOverflow =
    maxVisibleItems !== undefined &&
    maxVisibleItems >= 2 &&
    items.length > maxVisibleItems;
  const trailingStart = shouldOverflow
    ? items.length - (maxVisibleItems - 1)
    : 1;
  const overflowItems = shouldOverflow ? items.slice(1, trailingStart) : [];
  const visibleItems = shouldOverflow
    ? [items[0], ...items.slice(trailingStart)]
    : items;

  return (
    <CarbonBreadcrumb
      noTrailingSlash={noTrailingSlash}
      className={cn(className)}
    >
      {visibleItems.map((item, idx) => (
        <Fragment
          key={shouldOverflow && idx > 0 ? trailingStart + idx - 1 : idx}
        >
          {shouldOverflow && idx === 1 && (
            <CarbonBreadcrumbItem>
              <OverflowMenu
                size="sm"
                iconDescription="Show hidden breadcrumb items"
              >
                {overflowItems.map((overflowItem, overflowIdx) => (
                  <OverflowMenuItem
                    key={overflowIdx}
                    className={overflowItem.className}
                    href={overflowItem.href}
                    itemText={
                      overflowItem.overflowLabel ??
                      (typeof overflowItem.label === 'string'
                        ? overflowItem.label
                        : 'Breadcrumb')
                    }
                    onClick={overflowItem.onClick}
                  />
                ))}
              </OverflowMenu>
            </CarbonBreadcrumbItem>
          )}
          <CarbonBreadcrumbItem
            className={item.className}
            href={item.render || item.isCurrentPage ? undefined : item.href}
            aria-current={item.isCurrentPage ? 'page' : undefined}
          >
            {item.render ? item.render() : item.label}
          </CarbonBreadcrumbItem>
        </Fragment>
      ))}
    </CarbonBreadcrumb>
  );
}

export { CarbonBreadcrumbItem as BreadcrumbItem };
