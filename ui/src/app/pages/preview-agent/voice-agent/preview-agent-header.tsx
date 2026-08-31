import { FC } from 'react';
import { BrandedLogo } from '@/app/components/brand/branded-logo';
import { CustomerOptions } from '@/app/components/navigation/actionable-header';

export const PreviewAgentHeader: FC = () => (
  <header className="flex h-12 shrink-0 items-center justify-between border-b border-border-subtle bg-shell">
    <div className="flex min-w-0 items-center px-4">
      <BrandedLogo
        variant="full"
        className="h-6 w-auto max-w-[12rem] object-left"
        textClassName="text-base"
      />
    </div>
    <CustomerOptions showProjectSelector={false} />
  </header>
);
