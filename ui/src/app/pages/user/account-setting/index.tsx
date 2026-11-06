import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { Helmet } from '@/app/components/app-shell/helmet';
import { GhostButton, Tab } from '@/app/components/ui/primitives';
import { AccountSetting } from '@/app/pages/user/account-setting/account-setting';
import { NotificationSetting } from '@/app/pages/user/account-setting/notification-setting';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ChevronLeft } from '@carbon/icons-react';

export const AccountSettingPage = () => {
  const { goToDashboard } = useGlobalNavigation();
  return (
    <div className="flex flex-col flex-1 bg-surface h-full">
      <Helmet title="Account settings" />
      <PageHeaderBlock>
        <div className="flex items-center gap-1.5 min-w-0">
          <GhostButton
            size="sm"
            type="button"
            onClick={() => goToDashboard()}
            className="shrink-0"
          >
            <ChevronLeft size={16} data-testid="dashboard-back-icon" />
            Dashboard
          </GhostButton>
        </div>
      </PageHeaderBlock>
      <div className="flex min-h-0 flex-1 flex-col">
        <Tab
          strict={false}
          active="Account"
          className="sticky top-0 z-3 border-b border-border-subtle bg-surface "
          tabs={[
            {
              label: 'Account',
              element: <AccountSetting />,
            },
            {
              label: 'Notification',
              element: <NotificationSetting />,
            },
          ]}
        />
      </div>
    </div>
  );
};
