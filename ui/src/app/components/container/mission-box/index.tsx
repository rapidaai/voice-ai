import { ActionableHeader } from '@/app/components/navigation/actionable-header';
import { SidebarNavigation } from '@/app/components/navigation/sidebar';
import { Loader } from '@/app/components/loader';
import { useRapidaStore } from '@/hooks';
import { Toast } from '@/app/components/carbon/toast';
import { ProviderContextProvider } from '@/context/provider-context';
import { SidebarProvider } from '@/context/sidebar-context';

/**
 *
 * @param props
 * @returns
 */
export function MissionBox(props: { children?: any }) {
  useRapidaStore();
  return (
    <ProviderContextProvider>
      <SidebarProvider>
        <div className="flex h-[100dvh] relative w-[100dvw]">
          <SidebarNavigation />
          <main className="antialiased text-sm text-foreground relative bg-surface font-sans flex-1 flex w-full overflow-hidden">
            <div className="w-full flex flex-col h-full">
              <ActionableHeader />
              <div className="relative flex-1 overflow-hidden bg-surface flex flex-col">
                <div className="flex w-full absolute top-0 left-0 right-0 z-10">
                  <Loader />
                </div>
                <Toast />
                {props.children}
              </div>
            </div>
          </main>
        </div>
      </SidebarProvider>
    </ProviderContextProvider>
  );
}
