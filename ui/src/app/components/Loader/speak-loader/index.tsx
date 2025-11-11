import { cn } from '@/styles/media';
import React, { FC } from 'react';

export const SpeakLoader: FC<{ isRecording: boolean }> = ({ isRecording }) => {
  return (
    <div className="h-16 loader relative aspect-[1]">
      <div className="absolute z-10 top-0 right-0 left-0 bottom-0 flex justify-center items-center">
        <div
          className={cn(
            'flex items-center justify-center',
            'h-10 w-10  rounded-[2px] p-2',
            'text-white',
            isRecording ? 'bg-blue-500 text-white' : 'bg-blue-500 text-white',
          )}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth="1.2"
            stroke="currentColor"
            className="w-5 h-5 mx-auto "
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M19.114 5.636a9 9 0 0 1 0 12.728M16.463 8.288a5.25 5.25 0 0 1 0 7.424M6.75 8.25l4.72-4.72a.75.75 0 0 1 1.28.53v15.88a.75.75 0 0 1-1.28.53l-4.72-4.72H4.51c-.88 0-1.704-.507-1.938-1.354A9.009 9.009 0 0 1 2.25 12c0-.83.112-1.633.322-2.396C2.806 8.756 3.63 8.25 4.51 8.25H6.75Z"
            />
          </svg>
        </div>
      </div>
      {isRecording && (
        <>
          <div className="absolute animate-ripple-custom2s shadow-lg bg-blue-600/30 backdrop-blur-sm backdrop-opacity-20 z-2 rounded-[2px] border-[.1px] flex items-center border-gray-500/10! duration-[0.2s]! p-1 inset-[20%]"></div>
          <div className="absolute animate-ripple-custom3s shadow-lg bg-blue-600/20 backdrop-blur-sm backdrop-opacity-20 z-2 rounded-[2px] border-[.1px] flex items-center border-gray-500/10! duration-[0.2s]! p-1 inset-[10%]"></div>
          <div className="absolute animate-rippleCustom-4s shadow-lg bg-gray-600/10 z-1 rounded-[2px] border-[0.1px] flex items-center border-gray-600/10! duration-[0.4s]!  p-1 inset-[0%]"></div>
        </>
      )}
    </div>
  );
};
