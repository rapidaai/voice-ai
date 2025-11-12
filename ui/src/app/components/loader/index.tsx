import { LineLoader } from '@/app/components/loader/line-loader';
import { PageLoader } from '@/app/components/loader/page-loader';
import { useRapidaStore } from '@/hooks';

/**
 * General global loader
 * @returns
 */
export function Loader() {
  const { loadingType } = useRapidaStore();
  return loadingType === 'overlay' ? <PageLoader /> : <LineLoader />;
}
