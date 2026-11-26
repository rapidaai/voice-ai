import { FullPageError } from '@carbon/ibm-products/lib/components/FullPageError/FullPageError';
import { Link } from '@carbon/react';
import { GeneralFooter } from '@/app/components/layout/footer/general-footer';
import { PublicShellHeader } from '@/app/components/layout/public-shell/public-shell-header';

export type RouteErrorStatus = 403 | 404 | 500 | 503;

interface RouteErrorPageProps {
  status: RouteErrorStatus;
  actionLabel?: string;
  actionHref?: string;
  title?: string;
  description?: string;
}

const errorContent: Record<
  RouteErrorStatus,
  { title: string; description: string }
> = {
  403: {
    title: 'Access denied',
    description:
      'You are not authorized to access the requested page. Please verify that you are logged in to the hosting environment and your access permissions are correct.',
  },
  404: {
    title: 'Page not found',
    description:
      'The page you requested has moved or is unavailable, or the specified URL is not valid. Please check the URL or search the site for the requested content.',
  },
  500: {
    title: 'Something went wrong',
    description: "We couldn't complete your request. Try again.",
  },
  503: {
    title: 'Service unavailable',
    description: 'This service is temporarily unavailable. Try again shortly.',
  },
};

// FullPageError owns the main landmark, so render this outside MissionBox.
export function RouteErrorPage({
  status,
  actionLabel,
  actionHref,
  title = errorContent[status].title,
  description = errorContent[status].description,
}: RouteErrorPageProps) {
  return (
    <div className="flex h-dvh flex-col bg-surface text-foreground antialiased">
      <PublicShellHeader />
      <div className="min-h-0 flex-1 overflow-auto">
        <FullPageError
          kind={status === 403 || status === 404 ? `${status}` : 'custom'}
          label={`Error ${status}`}
          title={title}
          description={description}
        >
          {actionLabel && actionHref ? (
            <Link size="lg" href={actionHref}>
              – {actionLabel}
            </Link>
          ) : null}
        </FullPageError>
      </div>
      <GeneralFooter />
    </div>
  );
}
