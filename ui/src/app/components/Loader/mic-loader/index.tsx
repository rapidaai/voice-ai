import { cn } from '@/styles/media';
import React, { FC } from 'react';

export const MicLoader: FC<{ isRecording: boolean }> = ({ isRecording }) => {
  return (
    <div className="h-16 loader relative aspect-[1]">
      <div className="absolute z-10 top-0 right-0 left-0 bottom-0 flex justify-center items-center">
        <div
          className={cn(
            'flex items-center justify-center',
            'h-10 w-10  rounded-[2px] p-2',
            'text-white',
            isRecording ? 'bg-red-500 text-white' : 'bg-blue-500 text-white',
          )}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1.5}
            className="w-5 h-5 mx-auto "
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M12 18.75a6 6 0 0 0 6-6v-1.5m-6 7.5a6 6 0 0 1-6-6v-1.5m6 7.5v3.75m-3.75 0h7.5M12 15.75a3 3 0 0 1-3-3V4.5a3 3 0 1 1 6 0v8.25a3 3 0 0 1-3 3Z"
            />
          </svg>
        </div>
      </div>
      {isRecording && (
        <>
          <div className="absolute animate-ripple-custom2s shadow-lg bg-red-600/30 backdrop-blur-sm backdrop-opacity-20 z-2 rounded-[2px] border-[.1px] flex items-center border-gray-500/10! duration-[0.2s]! p-1 inset-[20%]"></div>
          <div className="absolute animate-ripple-custom3s shadow-lg bg-red-600/20 backdrop-blur-sm backdrop-opacity-20 z-2 rounded-[2px] border-[.1px] flex items-center border-gray-500/10! duration-[0.2s]! p-1 inset-[10%]"></div>
          <div className="absolute animate-rippleCustom-4s shadow-lg bg-gray-600/10 z-1 rounded-[2px] border-[0.1px] flex items-center border-gray-600/10! duration-[0.4s]!  p-1 inset-[0%]"></div>
        </>
      )}
    </div>
  );
};
