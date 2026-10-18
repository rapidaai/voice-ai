import { ActionableHeader } from '@/app/components/layout/navigation/actionable-header';
import { SidebarNavigation } from '@/app/components/layout/navigation/sidebar';
import { Loader } from '@/app/components/ui/feedback';
import { useRapidaStore } from '@/stores/app';
import { Toast } from '@/app/components/ui/feedback';
import { ProviderContextProvider } from '@/context/provider-context';
import { SidebarProvider } from '@/context/sidebar-context';
import type { HTMLAttributes } from 'react';
import { cn } from '@/utils';

export function MissionBox({
  children,
  className,
  ...attributes
}: HTMLAttributes<HTMLDivElement>) {
  useRapidaStore();
  return (
    <ProviderContextProvider>
      <SidebarProvider>
        <div
          {...attributes}
          className={cn('relative flex h-[100dvh] w-[100dvw]', className)}
        >
          <SidebarNavigation />
          <main className="relative flex min-w-0 flex-1 overflow-hidden bg-surface font-sans text-sm text-foreground antialiased">
            <div className="flex h-full w-full flex-col">
              <ActionableHeader />
              <div className="relative flex flex-1 flex-col overflow-hidden bg-surface">
                <div className="absolute left-0 right-0 top-0 z-10 flex w-full">
                  <Loader />
                </div>
                <Toast />
                {children}
              </div>
            </div>
          </main>
        </div>
      </SidebarProvider>
    </ProviderContextProvider>
  );
}
