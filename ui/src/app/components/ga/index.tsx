import { FC, useEffect } from 'react';
import ReactGA from 'react-ga4';
import { useLocation } from 'react-router-dom';

export const GA: FC = () => {
  const location = useLocation(); // Get current route

  useEffect(() => {
    // Enable analytics only in production and when not running on localhost
    if (
      process.env.NODE_ENV === 'production' &&
      window.location.hostname !== 'localhost'
    ) {
      ReactGA.initialize('G-SFF58VVY7H');
      ReactGA.send({ hitType: 'pageview', page: location.pathname });
    }
  }, [location.pathname]);

  return null; // No need to render anything with ReactGA
};
