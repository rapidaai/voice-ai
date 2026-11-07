import type { ReactNode } from 'react';
import type { CarbonIconType } from '@carbon/icons-react';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  HeaderGlobalBar,
} from '@carbon/react';

interface ResourceDetailHeaderProps {
  collectionHref: string;
  collectionLabel: string;
  resourceName: string;
  actionsLabel: string;
  primaryAction: {
    label: string;
    icon: CarbonIconType;
    onClick: () => void;
  };
  resourceMetadata?: ReactNode;
  secondaryActions?: ReactNode;
  testId?: string;
}

export function ResourceDetailHeader({
  collectionHref,
  collectionLabel,
  resourceName,
  actionsLabel,
  primaryAction,
  resourceMetadata,
  secondaryActions,
  testId,
}: ResourceDetailHeaderProps) {
  return (
    <header
      data-resource-detail-header
      data-testid={testId}
      className="flex h-12 shrink-0 items-center justify-between border-b border-border-subtle bg-shell pl-4"
    >
      <div className="flex min-w-0 items-center gap-3">
        <Breadcrumb noTrailingSlash>
          <BreadcrumbItem href={collectionHref}>
            {collectionLabel}
          </BreadcrumbItem>
          <BreadcrumbItem isCurrentPage>
            <span className="block max-w-[42vw] truncate">{resourceName}</span>
          </BreadcrumbItem>
        </Breadcrumb>
        {resourceMetadata}
      </div>
      <HeaderGlobalBar
        aria-label={actionsLabel}
        className="h-full items-center"
      >
        {secondaryActions}
        <Button
          aria-label={primaryAction.label}
          kind="primary"
          size="lg"
          renderIcon={primaryAction.icon}
          className="h-full! min-h-full! items-center justify-center whitespace-nowrap"
          onClick={primaryAction.onClick}
        >
          {primaryAction.label}
        </Button>
      </HeaderGlobalBar>
    </header>
  );
}
