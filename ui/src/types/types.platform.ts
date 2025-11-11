export {};

declare global {
  interface Window {
    isElectron: boolean;
    isFullscreen: boolean;
    api: {
      minimizeWindow: () => void;
      maximizeWindow: () => void;
      closeWindow: () => void;
      toggleFullscreen: () => void;
    };
  }
  namespace NodeJS {
    interface Process {
      type: 'browser' | 'renderer' | 'worker' | 'utility';
    }
  }
}
