import type { ReactNode } from 'react';
import { FeatureFlags } from '@carbon/react';

export function CarbonFeatureFlags({ children }: { children: ReactNode }) {
  return (
    <FeatureFlags
      enableV12Overflowmenu
      enableV12Autoalign
      enableV12DynamicFloatingStyles
    >
      {children}
    </FeatureFlags>
  );
}
