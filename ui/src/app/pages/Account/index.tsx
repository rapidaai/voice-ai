import { lazyLoad } from '@/utils/loadable';
import { LineLoader } from '@/app/components/Loader/line-loader';

export const AccountPersonalSettingPage = lazyLoad(
  () => import('./personal-setting'),
  module => module.PersonalSettingPage,
  {
    fallback: <LineLoader />,
  },
);

export const AccountNotificationSettingPage = lazyLoad(
  () => import('./NotificationSetting'),
  module => module.NotificationSettingPage,
  {
    fallback: <LineLoader />,
  },
);
