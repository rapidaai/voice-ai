import {
  useEffect,
  useRef,
  useState,
  type HTMLAttributes,
  type ReactNode,
} from 'react';
import { Text, type TypographyVariant } from '@/app/components/ui/primitives';
import { cn } from '@/utils';

type PageHeaderState = {
  isCollapsed: boolean;
};

const PAGE_HEADER_STICKY_TOOLBAR_CLASS_NAME =
  'sticky top-10 z-10 shrink-0 bg-layer';

export function PageHeaderStickyToolbar({
  children,
  className,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      {...attributes}
      data-page-header-sticky-toolbar
      className={`${PAGE_HEADER_STICKY_TOOLBAR_CLASS_NAME} ${className || ''}`}
    >
      {children}
    </div>
  );
}

interface PageHeaderProps {
  title: string;
  description?: string;
  eyebrow?: ReactNode;
  titleStart?: ReactNode;
  contextualActions?: ReactNode;
  pageActions?: ReactNode | ((state: PageHeaderState) => ReactNode);
  className?: string;
  contentClassName?: string;
  descriptionClassName?: string;
  titleClassName?: string;
  titleVariant?: TypographyVariant;
  showDivider?: boolean;
}

export function PageHeader({
  title,
  description,
  eyebrow,
  titleStart,
  contextualActions,
  pageActions,
  className,
  contentClassName,
  descriptionClassName,
  titleClassName,
  titleVariant,
  showDivider = true,
}: PageHeaderProps) {
  const collapseSentinel = useRef<HTMLSpanElement>(null);
  const [isCollapsed, setIsCollapsed] = useState(false);
  const currentPageActions =
    typeof pageActions === 'function'
      ? pageActions({ isCollapsed })
      : pageActions;

  useEffect(() => {
    if (!collapseSentinel.current || !window.IntersectionObserver) return;

    const observer = new IntersectionObserver(([entry]) => {
      setIsCollapsed(!entry.isIntersecting);
    });
    observer.observe(collapseSentinel.current);
    return () => observer.disconnect();
  }, []);

  return (
    <>
      <header
        data-page-header
        className={cn(
          'relative shrink-0 bg-layer',
          showDivider && 'border-b border-border-subtle',
          className,
        )}
      >
        <div
          data-page-header-content
          className={cn('relative px-5 py-6 md:px-8', contentClassName)}
        >
          <div className="flex flex-col gap-5 md:flex-row md:items-start md:justify-between">
            <div className="min-w-0 max-w-2xl">
              {eyebrow && <div className="mb-2">{eyebrow}</div>}
              <div className="flex flex-wrap items-center gap-2 text-foreground">
                {titleStart && (
                  <div data-page-header-icon className="shrink-0 pt-0.5">
                    {titleStart}
                  </div>
                )}
                <Text
                  as="h1"
                  variant={titleVariant ?? 'heading-04'}
                  className={cn(
                    !titleVariant &&
                      'text-[1.25rem]! leading-[1.75rem]! md:text-[1.75rem]! md:leading-[2.25rem]!',
                    titleClassName,
                  )}
                >
                  {title}
                </Text>
                {contextualActions}
              </div>
              {description && (
                <Text
                  as="div"
                  className={cn('mt-2 text-foreground', descriptionClassName)}
                  variant="body-compact-01"
                >
                  {description}
                </Text>
              )}
            </div>
            {currentPageActions && !isCollapsed && (
              <div className="w-full shrink-0 md:w-auto">
                {currentPageActions}
              </div>
            )}
          </div>
          <span
            ref={collapseSentinel}
            aria-hidden="true"
            className="pointer-events-none absolute inset-x-0 bottom-0 h-px"
          />
        </div>
      </header>

      <div
        data-page-header-sticky
        data-stuck={isCollapsed}
        className="sticky top-0 z-20 h-0"
      >
        <div
          data-page-header-sticky-content
          className={`h-10 border-b border-border-subtle bg-layer transition-[opacity,transform] duration-[240ms] ease-[cubic-bezier(0.2,0,0.38,0.9)] motion-reduce:transition-none ${
            isCollapsed
              ? 'visible translate-y-0 opacity-100'
              : 'invisible pointer-events-none -translate-y-4 opacity-0'
          }`}
        >
          <div
            data-page-header-sticky-container
            className={cn(
              'flex h-full items-center gap-4 px-5 md:px-8',
              contentClassName,
            )}
          >
            <div
              className={`${currentPageActions ? 'hidden md:flex' : 'flex'} min-w-0 items-center gap-2 text-foreground`}
            >
              {titleStart && (
                <div className="shrink-0 [&_svg]:h-5 [&_svg]:w-5">
                  {titleStart}
                </div>
              )}
              <Text
                variant="heading-compact-01"
                className={cn('truncate', titleClassName)}
              >
                {title}
              </Text>
            </div>
            {currentPageActions && isCollapsed && (
              <div className="ml-auto w-full shrink-0 md:w-auto">
                {currentPageActions}
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
}
