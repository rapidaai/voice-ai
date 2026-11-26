import { RouteErrorPage } from '@/app/components/ui/feedback';

export function PageNotFoundPage() {
  return (
    <RouteErrorPage
      status={404}
      actionLabel="Go to dashboard"
      actionHref="/dashboard"
    />
  );
}
