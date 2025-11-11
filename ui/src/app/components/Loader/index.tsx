import { LineLoader } from '@/app/components/Loader/line-loader';
import { PageLoader } from '@/app/components/Loader/page-loader';
import { useRapidaStore } from '@/hooks';

/**
 * General global loader
 * @returns
 */
export function Loader() {
  const { loadingType } = useRapidaStore();
  return loadingType === 'overlay' ? <PageLoader /> : <LineLoader />;
}
