import { ActionableHeader } from '@/app/components/navigation/actionable-header';
import { SidebarNavigation } from '@/app/components/navigation/sidebar';
import { Loader } from '@/app/components/Loader';
import { useRapidaStore } from '@/hooks';
import { Toast } from '@/app/components/toasts';
import { ProviderContextProvider } from '@/context/provider-context';

/**
 *
 * @param props
 * @returns
 */
export function MissionBox(props: { children?: any }) {
  const {} = useRapidaStore();
  return (
    <ProviderContextProvider>
      <main className="antialiased text-base text-gray-700 dark:text-gray-400 relative bg-[linear-gradient(103deg,var(--tw-gradient-stops))] from-custom-gray via-custom-pink to-custom-blue font-sans">
        <div className="flex w-full absolute top-0 left-0 right-0 z-10">
          <Loader />
        </div>
        <div className="flex h-screen relative w-full">
          <SidebarNavigation />
          <div className="w-full pl-14">
            <ActionableHeader />
            <div className="relative h-[calc(100dvh-3rem)] overflow-hidden dark:bg-gray-900 bg-light-background rounded-tl-md border-t border-l flex-1 flex flex-col">
              <Toast />
              {props.children}
            </div>
          </div>
          {/* <Agent
            agentConfig={
              new AgentConfig(
                '2219166493587800064',
                new InputOptions([Channel.Text], Channel.Text),
              )
            }
            connectionConfig={new AssistantConnectionConfig(
              ConnectionConfig.WithWebpluginClient({
                ApiKey:
                  '9eaaf1742be1ea0b3040ac34902e22b1182d3339127e939e499464715d472600',
                UserId: 'hello',
              }),
            ).withLocal()}
          >
            <Widget />
          </Agent> */}
        </div>
      </main>
    </ProviderContextProvider>
  );
}
