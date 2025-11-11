import { Loader } from '@/app/components/Loader';
import { useRapidaStore } from '@/hooks';
import { FC, HTMLAttributes } from 'react';

export const Box: FC<HTMLAttributes<HTMLDivElement>> = props => {
  const {} = useRapidaStore();

  return (
    <main className="antialiaseddark:text-gray-400 relative">
      {/* bg-gray-100 dark:bg-gray-800 */}
      <div className="flex w-full">
        <Loader />
      </div>
      <div className="flex h-screen relative w-full">{props.children}</div>
    </main>
  );
};
