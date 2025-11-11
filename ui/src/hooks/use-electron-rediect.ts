import { useEnvironment } from '@/context/environment-context';
import { useNavigate } from 'react-router-dom';

export const useElectronRedirect = () => {
  const navigate = useNavigate();
  const { isElectron } = useEnvironment();

  const redirecting = to => {
    if (isElectron) {
      return to;
    }
    return `/preview${to}`;
  };

  const goTo = (to: string) => navigate(redirecting(to));
  return {
    redirecting,
    goTo,
  };
};
