import {
  Header,
  HeaderMenuItem,
  HeaderName,
  HeaderNavigation,
} from '@carbon/react';
import { useTheme } from '@/theme/theme-provider';

export function GeneralFooter() {
  const { theme } = useTheme();

  return (
    <Header
      aria-label={`${theme.brand.name} Platform`}
      className="[inset-block-start:auto]! [inset-block-end:0]! border-gray-200! dark:border-gray-900! border-t!"
    >
      <HeaderName href="#" prefix={theme.brand.name}>
        [Platform]
      </HeaderName>
      <HeaderNavigation
        aria-label={`${theme.brand.name} [Platform]`}
        className=""
      >
        <HeaderMenuItem href={theme.links.terms}>
          <span className="opacity-80">Terms and Conditions</span>
        </HeaderMenuItem>
        <HeaderMenuItem href={theme.links.privacy}>
          <span className="opacity-80">Privacy Policy</span>
        </HeaderMenuItem>
        <HeaderMenuItem href={theme.links.documentation}>
          <span className="opacity-80">Documentation</span>
        </HeaderMenuItem>
        <HeaderMenuItem href={theme.links.source}>
          <span className="opacity-80">Source</span>
        </HeaderMenuItem>
        <HeaderMenuItem href={theme.links.support}>
          <span className="opacity-80">Support</span>
        </HeaderMenuItem>
      </HeaderNavigation>
    </Header>
  );
}
