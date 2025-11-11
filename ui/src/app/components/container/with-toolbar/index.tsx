import { Toast } from '@/app/components/toasts';
import { ChevronsLeftRightIcon } from '@/app/components/Icon/chevrons-left-right';
import { ChevronsRightLeftIcon } from '@/app/components/Icon/chevrons-right-left';
import { CloseIcon } from '@/app/components/Icon/Close';
import { MinusIcon } from '@/app/components/Icon/minus';
import { Loader } from '@/app/components/Loader';
import { useRapidaStore } from '@/hooks';
import { FC, HTMLAttributes } from 'react';

export const WithToolbar: FC<HTMLAttributes<HTMLDivElement>> = props => {
  const {} = useRapidaStore();
  const handleMinimize = () => {
    window.api.minimizeWindow();
  };

  const handleMaximize = () => {
    window.api.maximizeWindow();
  };

  const handleClose = () => {
    window.api.closeWindow();
  };

  const fullWindow = () => {
    window.api.toggleFullscreen();
  };
  const { loading } = useRapidaStore();

  return (
    <div className=" bg-[linear-gradient(103deg,var(--tw-gradient-stops))] from-custom-gray via-custom-pink to-custom-blue">
      <div className="flex overflow-hidden h-screen flex-col relative">
        <Toast />
        {process.env.NODE_ENV !== 'production' && (
          <div className="flex right-2 top-2 absolute space-x-1">
            <div className="rounded-[2px] w-fit px-2 bg-amber-600/10 text-amber-700 font-medium">
              {process.env.NODE_ENV}
            </div>
            {window.isElectron && (
              <div className="rounded-[2px] w-fit px-2 bg-blue-600/10 text-blue-700 font-medium">
                electron-app
              </div>
            )}
          </div>
        )}
        <div className="h-10 p-3">
          {window.isElectron && (
            <div className="flex items-center space-x-2 mr-4 group">
              <button
                onClick={handleClose}
                className="w-3 h-3 flex items-center justify-center rounded-[2px] bg-red-500 hover:bg-red-600 transition duration-300"
                aria-label="Close"
              >
                <CloseIcon
                  className="w-2.5 h-2.5 block text-black group-hover:opacity-100 opacity-0"
                  strokeWidth={1.5}
                />
              </button>
              <button
                onClick={handleMinimize}
                className="w-3 h-3 flex items-center justify-center rounded-[2px] bg-yellow-500 hover:bg-yellow-600 transition duration-300"
                aria-label="Minimize"
              >
                <MinusIcon
                  className="w-2.5 h-2.5 block text-black group-hover:opacity-100 opacity-0"
                  strokeWidth={1.5}
                />
              </button>
              <button
                onClick={fullWindow}
                className="w-3 h-3 flex items-center justify-center rounded-[2px] bg-green-500 hover:bg-green-600 transition duration-300"
                aria-label="Maximize"
              >
                {window.isFullscreen ? (
                  <ChevronsRightLeftIcon
                    className="w-2.5 h-2.5 block text-black group-hover:opacity-100 opacity-0 -rotate-45"
                    strokeWidth={1.5}
                  />
                ) : (
                  <ChevronsLeftRightIcon
                    className="w-2.5 h-2.5 block text-black group-hover:opacity-100 opacity-0 -rotate-45"
                    strokeWidth={1.5}
                  />
                )}
              </button>
            </div>
          )}
        </div>
        <Loader />
        {props.children}
      </div>
    </div>
  );
};
