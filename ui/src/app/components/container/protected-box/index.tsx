import React, { useContext, useEffect } from 'react';
import { Navigate, useLocation, useSearchParams } from 'react-router-dom';
import { AuthContext } from '@/context/auth-context';
/**
 *
 * @param param0
 * @returns
 */
export function ProtectedBox(props: {
  children: React.ReactElement;
  allowedRoles?: string[];
}) {
  /**
   * current pathname
   */
  const { pathname, search } = useLocation();
  /**
   * authentication context with a setter
   */
  const { isAuthenticated, isThereOrganization, isThereProject } =
    useContext(AuthContext);
  /**
   * if it is not authenticated then signin redirect
   */
  if (isAuthenticated && !isAuthenticated()) {
    return <Navigate to={`/auth/signin${search}`} />;
  }

  /**
   * organization onboarding
   */
  if (pathname === '/onboarding/organization') {
    return props.children;
  }

  /**
   * if organization is not there the redirect to organization
   */
  if (isThereOrganization && !isThereOrganization())
    return <Navigate to="/onboarding/organization" />;

  if (pathname === '/onboarding/project') {
    return props.children;
  }

  /**
   * if there is no project then
   */
  if (isThereProject && !isThereProject())
    return <Navigate to="/onboarding/project" />;

  return props.children;
}

export function IgnoreBox(props: { children: React.ReactElement }) {
  //   /**
  //    *
  //    */
  const [searchParams] = useSearchParams();
  const searchParamMap = Object.fromEntries(searchParams.entries());
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

    const isExternalAuthValid = () => {
      return (
        searchParamMap['next'] &&
        searchParamMap['externalValidation'] &&
        isAuthValid()
      );
    };

    if (isExternalAuthValid()) {
      window.location.replace(searchParamMap['next']);
      return;
    }
    if (unauthenticate) unauthenticate();
  }, []);
  return props.children;
}
