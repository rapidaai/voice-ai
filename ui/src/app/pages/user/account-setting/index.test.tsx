import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AccountSettingPage } from './index';

const mockGoToDashboard = jest.fn();

jest.mock('@/app/components/app-shell/helmet', () => ({
  Helmet: ({ title }: any) => <div data-testid="helmet">{title}</div>,
}));

jest.mock('@/app/components/layout/blocks/page-header-block', () => ({
  PageHeaderBlock: ({ children }: any) => <header>{children}</header>,
}));

jest.mock('@/app/components/ui/primitives/tabs', () => ({
  Tab: ({ active, tabs }: any) => (
    <div data-active-tab={active}>
      {tabs.map((tab: any) => (
        <section key={tab.label}>{tab.element}</section>
      ))}
    </div>
  ),
}));

jest.mock('@/app/pages/user/account-setting/account-setting', () => ({
  AccountSetting: () => <div>Account content</div>,
}));

jest.mock('@/app/pages/user/account-setting/notification-setting', () => ({
  NotificationSetting: () => <div>Notification content</div>,
}));

jest.mock('@/hooks/use-global-navigator', () => ({
  useGlobalNavigation: () => ({
    goToDashboard: mockGoToDashboard,
  }),
}));

jest.mock('@carbon/icons-react', () => ({
  ChevronLeft: (props: any) => <svg {...props} />,
}));

describe('AccountSettingPage', () => {
  beforeEach(() => {
    mockGoToDashboard.mockClear();
  });

  it('renders account settings tabs with the Carbon back icon', () => {
    render(<AccountSettingPage />);

    expect(screen.getByText('Account content')).toBeInTheDocument();
    expect(screen.getByText('Notification content')).toBeInTheDocument();
    expect(screen.getByTestId('dashboard-back-icon')).toBeInTheDocument();
  });

  it('navigates to the dashboard from the back action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<AccountSettingPage />);

    await user.click(screen.getByRole('button', { name: 'Dashboard' }));

    expect(mockGoToDashboard).toHaveBeenCalledTimes(1);
  });
});
