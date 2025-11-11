import { ToolIcon } from '@/app/components/Icon/tool';
import { SideTab, SideTabLink } from '@/app/components/tab-link';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { BarChart, Bolt, ChevronLeft, Package, Webhook } from 'lucide-react';
import { FC, HTMLAttributes } from 'react';
import { Outlet, useParams } from 'react-router-dom';

export const AssistantManageLayout: FC<HTMLAttributes<HTMLDivElement>> = () => {
  const { assistantId } = useParams();
  const { goToAssistant } = useGlobalNavigation();
  return (
    <div className="flex-1 flex relative grow h-full overflow-hidden">
      <aside
        className="w-80 border-r bg-white dark:bg-gray-900 z-1 overflow-auto shrink-0"
        aria-label="Sidebar"
      >
        <ul className="text-sm">
          <li>
            <SideTab
              to="#"
              onClick={() => {
                goToAssistant(assistantId!);
              }}
              className="hover:text-red-600 hover:border-b hover:border-b-red-600 text-red-600 border-b h-10 flex items-center"
            >
              <ChevronLeft className="w-5 h-5 mr-1" strokeWidth={1.5} />
              <span className="">Back to assistant</span>
            </SideTab>
          </li>

          <li>
            <SideTabLink to="deployment/" className="h-11">
              <Package className="w-4 h-4 mr-2" strokeWidth={1.5} /> Deployment
            </SideTabLink>
          </li>

          <li>
            <SideTabLink to="configure-tool" className="h-11">
              <ToolIcon className="w-4 h-4 mr-2" strokeWidth={1.5} /> Tools and
              MCP
            </SideTabLink>
          </li>
          <li>
            <SideTabLink to="configure-analysis" className="h-11">
              <BarChart className="w-4 h-4 mr-2" /> Analysis
            </SideTabLink>
          </li>
          <li>
            <SideTabLink to="configure-webhook" className="h-11">
              <Webhook className="w-4 h-4 mr-2" strokeWidth={1.5} />
              Webhooks
            </SideTabLink>
          </li>
          <li>
            <SideTabLink to="edit-assistant" className="h-11">
              <Bolt className="w-4 h-4 mr-2" strokeWidth={1.5} />
              Settings
            </SideTabLink>
          </li>
        </ul>
      </aside>
      <Outlet />
    </div>
  );
};
