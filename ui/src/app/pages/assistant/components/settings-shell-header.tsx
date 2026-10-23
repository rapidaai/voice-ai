import { Add, Renew, type CarbonIconType } from '@carbon/icons-react';
import {
  Button,
  Header,
  HeaderGlobalAction,
  HeaderGlobalBar,
  HeaderName,
} from '@carbon/react';
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
    <Header
      aria-label={`${title} page header`}
      className="relative! inset-auto! z-auto! h-12! min-h-12! shrink-0 border-b! border-border-subtle! bg-[var(--cds-background)]! p-0!"
    >
      <HeaderName href={href} prefix="" className="h-full">
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
    </Header>
  );
}
