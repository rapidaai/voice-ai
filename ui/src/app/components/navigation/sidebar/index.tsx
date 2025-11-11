import { Observability } from '@/app/components/navigation/sidebar/observability';
import { Deployment } from '@/app/components/navigation/sidebar/deployment';
import { Dashboard } from '@/app/components/navigation/sidebar/dashboard';
import { Team } from '@/app/components/navigation/sidebar/team';
import { Project } from '@/app/components/navigation/sidebar/project';
import { Vault } from '@/app/components/navigation/sidebar/vault';
import { SidebarLabel } from '@/app/components/navigation/sidebar/sidebar-label';
import { Knowledge } from '@/app/components/navigation/sidebar/knowledge';
import { Aside } from '@/app/components/aside';
import { ExternalTool } from '@/app/components/navigation/sidebar/external-tools';
import { useWorkspace } from '@/context/workplace-context';

/**
 *
 * @param props
 * @returns
 */
export function SidebarNavigation(props: {}) {
  const workspace = useWorkspace();
  return (
    <Aside className="space-y-2 ">
      <div className="flex rounded-[2px] my-2 text-blue-600 items-center">
        <div className="pl-[0.8rem] py-2.5 shrink-0">{workspace.logo}</div>
      </div>
      <div className="space-y-4">
        <ul className="space-y-1">
          <Dashboard />
          <Observability />
          <Deployment />
          <Knowledge />
        </ul>
        <div className="space-y-3">
          <SidebarLabel className="uppercase truncate pl-3 text-xs font-medium opacity-0 group-hover:opacity-100">
            Integrations
          </SidebarLabel>
          <ul className="space-y-1">
            <ExternalTool />
            <Vault />
          </ul>
        </div>
        <div className="space-y-3">
          <SidebarLabel className="uppercase truncate pl-3 text-xs font-medium opacity-0 group-hover:opacity-100">
            Organizations
          </SidebarLabel>
          <ul className="space-y-1">
            {/* <Organization /> */}
            <Team />
            <Project />
          </ul>
        </div>
      </div>
    </Aside>
  );
}
