import { Play } from '@carbon/icons-react';
import { Button, HeaderGlobalBar, Loading } from '@carbon/react';
import { FC } from 'react';

export const PlaygroundHeader: FC<{
  isValid: boolean;
  loading: boolean;
  disabled?: boolean;
}> = ({ loading, disabled = false }) => {
  return (
    <header
      className="flex h-12 shrink-0 items-center justify-between border-b border-border-subtle bg-layer"
      data-testid="endpoint-page-header"
    >
      <div className="min-w-0 pl-4">
        <h1 className="truncate text-sm font-semibold text-foreground">
          Playground
        </h1>
      </div>

      <HeaderGlobalBar
        aria-label="Endpoint playground actions"
        className="h-full items-center"
      >
        <Button
          type="submit"
          kind="primary"
          size="lg"
          disabled={loading || disabled}
          renderIcon={!loading ? Play : undefined}
          className="h-full! min-h-full! items-center justify-center"
        >
          {loading ? 'Running' : 'Run'}
          {loading && (
            <Loading
              description="Executing endpoint"
              withOverlay={false}
              small
              className="ml-2"
            />
          )}
        </Button>
      </HeaderGlobalBar>
    </header>
  );
};
