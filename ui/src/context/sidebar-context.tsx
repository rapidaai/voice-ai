import * as React from 'react';

const SIDEBAR_COOKIE_NAME = 'sidebar:state';

type SidebarContextType = {
  open: boolean;
  setOpen: (open: boolean) => void;
  toggleSidebar: () => void;
  locked: boolean;
  setLocked: (locked: boolean) => void;
};

const SidebarContext = React.createContext<SidebarContextType | null>(null);

function isMobileViewport() {
  return (
    typeof window !== 'undefined' &&
    window.matchMedia?.('(max-width: 767px)').matches
  );
}

function useSidebar() {
  const context = React.useContext(SidebarContext);
  if (!context) {
    throw new Error('useSidebar must be used within a SidebarProvider.');
  }

  return context;
}

const SidebarProvider = React.forwardRef<
  HTMLDivElement,
  React.ComponentProps<'div'> & {
    defaultOpen?: boolean;
    open?: boolean;
    onOpenChange?: (open: boolean) => void;
  }
>(
  (
    {
      defaultOpen = localStorage.getItem(SIDEBAR_COOKIE_NAME) !== 'false',
      open: openProp,
      onOpenChange: setOpenProp,
      className,
      style,
      children,
      ...props
    },
    ref,
  ) => {
    const [_open, _setOpen] = React.useState(
      () => defaultOpen && !isMobileViewport(),
    );
    const open = openProp ?? _open;

    const [locked, setLockedState] = React.useState(() => {
      return (
        !isMobileViewport() &&
        localStorage.getItem(SIDEBAR_COOKIE_NAME) !== 'false'
      );
    });

    React.useEffect(() => {
      const media = window.matchMedia?.('(max-width: 767px)');
      if (!media) return undefined;
      const collapseOnMobile = (event: MediaQueryListEvent) => {
        if (!event.matches) return;
        setLockedState(false);
        _setOpen(false);
      };
      media.addEventListener('change', collapseOnMobile);
      return () => media.removeEventListener('change', collapseOnMobile);
    }, []);

    const setLocked = React.useCallback((value: boolean) => {
      setLockedState(value);

      // Persist the locked state in a cookie
      localStorage.setItem(SIDEBAR_COOKIE_NAME, value ? 'true' : 'false');
      _setOpen(value); // Force open when locked
    }, []);

    const setOpen = React.useCallback(
      (value: boolean | ((value: boolean) => boolean)) => {
        const openState = typeof value === 'function' ? value(open) : value;

        if (locked && !openState) {
          console.warn('Cannot close the sidebar when it is locked.');
          return;
        }

        if (setOpenProp) {
          setOpenProp(openState);
        } else {
          _setOpen(openState);
        }
      },
      [setOpenProp, open, locked],
    );

    const toggleSidebar = React.useCallback(() => {
      setOpen(open => !open);
    }, [setOpen]);

    const contextValue = React.useMemo<SidebarContextType>(
      () => ({
        open,
        setOpen,
        toggleSidebar,
        locked,
        setLocked,
      }),
      [open, setOpen, toggleSidebar, locked, setLocked],
    );

    return (
      <SidebarContext.Provider value={contextValue}>
        {children}
      </SidebarContext.Provider>
    );
  },
);
SidebarProvider.displayName = 'SidebarProvider';

export { SidebarProvider, useSidebar };
export type { SidebarContextType as SidebarContext };
