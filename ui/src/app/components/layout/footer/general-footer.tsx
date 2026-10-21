import type { HTMLAttributes } from 'react';
import { HeaderMenuItem, HeaderName, HeaderNavigation } from '@carbon/react';
import { Text } from '@/app/components/ui/primitives/text';
import { useTheme } from '@/theme/theme-provider';
import { cn } from '@/utils';

const footerMenuItemClassName = '[&>a]:bg-transparent!';

export function GeneralFooter({
  className,
  ...attributes
}: HTMLAttributes<HTMLElement>) {
  const { theme } = useTheme();

  return (
    <footer
      {...attributes}
      aria-label={`${theme.brand.name} platform footer`}
      className={cn(
        'h-12 flex shrink-0 items-center',
        'bg-shell text-foreground',
        'border-t border-border-subtle',
        className,
      )}
    >
      <HeaderName href="/" prefix="">
        <Text variant="heading-compact-01">{theme.brand.name} Platform</Text>
      </HeaderName>
      <HeaderNavigation aria-label={`${theme.brand.name} footer links`}>
        <HeaderMenuItem
          className={footerMenuItemClassName}
          href={theme.links.terms}
        >
          <Text variant="body-compact-01">Terms and Conditions</Text>
        </HeaderMenuItem>
        <HeaderMenuItem
          className={footerMenuItemClassName}
          href={theme.links.privacy}
        >
          <Text variant="body-compact-01">Privacy Policy</Text>
        </HeaderMenuItem>
        <HeaderMenuItem
          className={footerMenuItemClassName}
          href={theme.links.documentation}
        >
          <Text variant="body-compact-01">Documentation</Text>
        </HeaderMenuItem>
        <HeaderMenuItem
          className={footerMenuItemClassName}
          href={theme.links.source}
        >
          <Text variant="body-compact-01">Source</Text>
        </HeaderMenuItem>
        <HeaderMenuItem
          className={footerMenuItemClassName}
          href={theme.links.support}
        >
          <Text variant="body-compact-01">Support</Text>
        </HeaderMenuItem>
      </HeaderNavigation>
    </footer>
  );
}
