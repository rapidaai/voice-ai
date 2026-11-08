import { FC } from 'react';
import { BrandedLogo } from '@/app/components/layout/brand/branded-logo';
import { CustomerOptions } from '@/app/components/layout/navigation/actionable-header';
import { CompactPageHeader } from '@/app/components/layout/compact-page-header';

export const PreviewAgentHeader: FC = () => (
  <CompactPageHeader className="pl-0">
    <div className="flex min-w-0 items-center px-4">
      <BrandedLogo
        variant="full"
        className="h-6 w-auto max-w-[12rem] object-left"
        textClassName="text-base"
      />
    </div>
    <CustomerOptions showProjectSelector={false} />
  </CompactPageHeader>
);
