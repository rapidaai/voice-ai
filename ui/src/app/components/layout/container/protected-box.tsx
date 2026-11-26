import React, { useContext, useEffect } from 'react';
import { Navigate, useLocation, useSearchParams } from 'react-router-dom';
import { AuthContext } from '@/context/auth-context';

export function ProtectedBox({ children }: { children: React.ReactElement }) {
  const { pathname, search } = useLocation();
  const { hasOrganizations, isAuthenticated } = useContext(AuthContext);

  if (isAuthenticated && !isAuthenticated()) {
    return <Navigate to={`/auth/signin${search}`} />;
  }

  if (pathname === '/onboarding/organization') {
    return children;
  }

  if (hasOrganizations && !hasOrganizations())
    return <Navigate to="/onboarding/organization" />;

  if (pathname === '/onboarding/project') {
    return children;
  }

  return children;
}

interface IgnoreBoxProps {
  children: React.ReactElement;
  replaceLocation?: (url: string) => void;
}

const replaceWindowLocation = (url: string) => {
  window.location.replace(url);
};

export function IgnoreBox({
  children,
  replaceLocation = replaceWindowLocation,
}: IgnoreBoxProps) {
  const [searchParams] = useSearchParams();
  const nextUrl = searchParams.get('next');
  const externalValidation = searchParams.get('externalValidation');
  const {
    unauthenticate,
    isAuthenticated,
    isThereOrganization,
    isThereProject,
  } = useContext(AuthContext);

  useEffect(() => {
    const isAuthValid = () =>
      isAuthenticated &&
      isAuthenticated() &&
      isThereOrganization &&
      isThereOrganization() &&
      isThereProject &&
      isThereProject();

    if (nextUrl && externalValidation && isAuthValid()) {
      replaceLocation(nextUrl);
      return;
    }
    if (unauthenticate) unauthenticate();
  }, [
    externalValidation,
    isAuthenticated,
    isThereOrganization,
    isThereProject,
    nextUrl,
    replaceLocation,
    unauthenticate,
  ]);
  return children;
}
