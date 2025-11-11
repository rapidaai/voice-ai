import { ErrorContainer } from '@/app/components/error-container';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { FC } from 'react';

export const PageNotFoundPage: FC<{}> = () => {
  const { goToDashboard } = useGlobalNavigation();
  return (
    <div className="h-screen w-screen flex items-center">
      <ErrorContainer
        onAction={goToDashboard}
        code="404"
        actionLabel="Go back"
        title={"Sorry we couldn't find this page."}
        description="But dont worry, you can find plenty of other things on our homepage."
      />
    </div>
  );
};
