import { Add, Renew, type CarbonIconType } from '@carbon/icons-react';
import {
  Button,
  HeaderGlobalAction,
  HeaderGlobalBar,
  HeaderName,
} from '@carbon/react';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { Text } from '@/app/components/ui/primitives';

interface AssistantSettingsShellHeaderProps {
  title: string;
  href: string;
  onReload?: () => void;
  reloadLabel?: string;
  primaryAction?: {
    label: string;
    onClick: () => void;
    icon?: CarbonIconType;
  };
}

export function AssistantSettingsShellHeader({
  title,
  href,
  onReload,
  reloadLabel = 'Reload',
  primaryAction,
}: AssistantSettingsShellHeaderProps) {
  return (
    <PageHeaderBlock aria-label={`${title} page header`}>
      <HeaderName href={href} prefix="" className="h-full px-0!">
        <span role="heading" aria-level={1} className="block min-w-0 truncate">
          <Text as="span" variant="heading-compact-01">
            {title}
          </Text>
        </span>
      </HeaderName>
      {(onReload || primaryAction) && (
        <HeaderGlobalBar
          aria-label={`${title} header actions`}
          className="h-full items-center"
        >
          {onReload && (
            <HeaderGlobalAction
              aria-label={reloadLabel}
              tooltipAlignment="end"
              onClick={onReload}
            >
              <Renew size={16} />
            </HeaderGlobalAction>
          )}
          {primaryAction && (
            <Button
              kind="primary"
              size="lg"
              renderIcon={primaryAction.icon ?? Add}
              className="h-full! min-h-full! whitespace-nowrap"
              onClick={primaryAction.onClick}
            >
              {primaryAction.label}
            </Button>
          )}
        </HeaderGlobalBar>
      )}
    </PageHeaderBlock>
  );
}
