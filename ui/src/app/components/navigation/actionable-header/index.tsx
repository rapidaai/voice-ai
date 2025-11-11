import React, { useState, useContext, useEffect, FC } from 'react';
import { HelpPopover } from '@/app/components/Popover/HelpPopover';
import { ProjectSelectorDropdown } from '@/app/components/Dropdown/ProjectDropdown';
import { cn } from '@/styles/media';
import { ProfilePopover } from '@/app/components/Popover/ProfilePopover';
import { IButton } from '@/app/components/Form/Button';
import { useLocation } from 'react-router-dom';
import { CustomLink } from '@/app/components/custom-link';
import { useDarkMode } from '@/context/dark-mode-context';
import { AuthContext } from '@/context/auth-context';
import { TextImage } from '@/app/components/Image/TextImage';
import { ChevronRight, HelpCircle, Moon, Sun } from 'lucide-react';
/**
 *
 * @param props
 * @returns
 */
export function ActionableHeader(props: { reload?: boolean }) {
  const location = useLocation();
  const { pathname } = location;
  const [breadcrumbs, setBreadcrumbs] = useState<
    { label: string; href: string }[]
  >([]);

  useEffect(() => {
    const pathParts = pathname.split('/').filter(part => part?.trim() !== '');
    setBreadcrumbs(
      pathParts?.map((part, partIndex) => {
        const previousParts = pathParts.slice(0, partIndex);
        return {
          label: part,
          href:
            previousParts?.length > 0
              ? `/${previousParts?.join('/')}/${part}`
              : `/${part}`,
        };
      }) || [],
    );
  }, [pathname]);
  return (
    <header className={cn('antialiased')}>
      <div className="flex items-center justify-between pl-4">
        <ol className="flex items-center truncate">
          {breadcrumbs.map((x, idx) => {
            return <BreadcrumbElement label={x} key={idx} />;
          })}
        </ol>
        <CustomerOptions />
      </div>
    </header>
  );
}

function BreadcrumbElement(props: { label: { href: string; label: string } }) {
  return (
    <>
      <li>
        <CustomLink
          className="capitalize hover:text-blue-600 font-medium"
          to={props.label.href}
        >
          {props.label.label}
        </CustomLink>
      </li>

      <li className="px-1.5 last:hidden">
        <ChevronRight className="w-4 h-4" strokeWidth={1.5} />
      </li>
    </>
  );
}

export const CustomerOptions: FC<{ placement?: 'top' | 'bottom' }> = ({
  placement,
}) => {
  /**
   * Current authentication information
   */
  const {
    currentUser,
    projectRoles,
    currentProjectRole,
    setCurrentProjectRole,
  } = useContext(AuthContext);

  const [accountDropdownOpen, setAccountDropdownOpen] = useState(false);
  const [helpDropdownOpen, setHelpDropdownOpen] = useState(false);
  const { isDarkMode, toggleDarkMode } = useDarkMode();

  return (
    <>
      <div className={cn('flex items-center')}>
        <div className="border-l border-r">
          {projectRoles && setCurrentProjectRole && (
            <ProjectSelectorDropdown
              projects={projectRoles}
              project={currentProjectRole}
              setProject={setCurrentProjectRole}
              placement={placement}
            />
          )}
        </div>

        <div className="relative inline-flex">
          <IButton
            className={cn(
              'h-12 w-12 border-r',
              helpDropdownOpen && 'bg-gray-200 dark:bg-gray-800',
            )}
            onClick={() => setHelpDropdownOpen(!helpDropdownOpen)}
          >
            <span className="sr-only">Need help?</span>
            <HelpCircle strokeWidth={1.5} />
          </IButton>
          <HelpPopover
            align={placement ? placement : 'bottom-end'}
            open={helpDropdownOpen}
            setOpen={setHelpDropdownOpen}
          />
        </div>

        {/* when will impliment the dark and light theme */}
        <IButton className={'h-12 w-12 border-r'} onClick={toggleDarkMode}>
          <Sun
            className={cn(isDarkMode ? 'hidden' : 'block')}
            strokeWidth={1.5}
          />
          <Moon
            className={cn(!isDarkMode ? 'hidden' : 'block')}
            strokeWidth={1.5}
          />
          <span className="sr-only">
            Switch to {isDarkMode ? 'light' : 'dark'} mode
          </span>
        </IButton>

        <div className="relative inline-flex">
          <IButton
            className={cn(
              'h-12 w-12 border-r',
              accountDropdownOpen && 'bg-gray-200 dark:bg-gray-800',
            )}
            aria-haspopup="true"
            onClick={() => setAccountDropdownOpen(!accountDropdownOpen)}
            aria-expanded={accountDropdownOpen}
          >
            <span className="sr-only">Account</span>
            <TextImage name={currentUser?.name!} size={8} />
          </IButton>
          <ProfilePopover
            align={placement ? `${placement}-start` : 'bottom-end'}
            open={accountDropdownOpen}
            setOpen={setAccountDropdownOpen}
            account={{ email: currentUser ? currentUser.email : '' }}
          />
        </div>
      </div>
    </>
  );
};
