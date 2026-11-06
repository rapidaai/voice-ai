import {
  StaticTermsPage,
  StaticPrivacyPage,
  StaticPageNotFoundPage,
} from '@/app/pages/static-pages';
import type { RouteObject } from 'react-router-dom';

export const staticRoute: RouteObject = {
  path: '/static',
  children: [
    { path: 'privacy-policy', element: <StaticPrivacyPage /> },
    { path: 'terms-conditions', element: <StaticTermsPage /> },
    { path: '*', element: <StaticPageNotFoundPage /> },
  ],
};
