import {
  StaticTermsPage,
  StaticPrivacyPage,
  StaticPageNotFoundPage,
} from '@/app/pages/static-pages';
import React from 'react';
import { Routes, Route } from 'react-router-dom';

export function StaticRoute() {
  return (
    <Routes>
      <Route
        key="/static/privacy-policy"
        path="privacy-policy"
        element={<StaticPrivacyPage />}
      />
      <Route
        key="/static/terms-conditions"
        path="terms-conditions"
        element={<StaticTermsPage />}
      />
      <Route path="*" element={<StaticPageNotFoundPage />} />
    </Routes>
  );
}
