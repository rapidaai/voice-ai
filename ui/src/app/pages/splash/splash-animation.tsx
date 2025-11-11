import { Helmet } from '@/app/components/Helmet';
import { AnimatedLine } from '@/app/components/Loader/line-loader';
import { ClickableLogo } from '@/app/components/Logo/ClickableLogo';
import { FC, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

export const SplashAnimationPage: FC = () => {
  const navigation = useNavigate();

  useEffect(() => {
    setTimeout(() => {
      navigation('/dashboard');
    }, 3000);
  }, []);

  return (
    <>
      <Helmet title="Welcome to rapid.ai" />
      <div className="h-screen w-full flex items-center justify-center relative">
        <div className="absolute top-0 right-0 left-0">
          <AnimatedLine animate="infinite" />
        </div>
        <div>
          <ClickableLogo
            className="h-12"
            darkLogo={'./images/logos/logo-04.png'}
            lightLogo={'./images/logos/logo-02.png'}
          />
        </div>
      </div>
    </>
  );
};
