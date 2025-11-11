import { FC, HTMLAttributes, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import {
  ChevronDown,
  CircleFadingPlus,
  MessageSquareText,
  Mic,
  X,
} from 'lucide-react';

import {
  useConnectAgent,
  MultibandAudioVisualizerComponent,
  useMultibandMicrophoneTrackVolume,
  useSelectInputDeviceAgent,
  useInputModeToggleAgent,
  VoiceAgent,
} from '@rapidaai/react';
import { cn } from '@/styles/media';
import { Tooltip } from '@/app/components/base/tooltip';

/**
 *
 */
interface AudioMessagingActionProps extends HTMLAttributes<HTMLDivElement> {
  voiceAgent: VoiceAgent;
  placeholder?: string;
}

/**
 *
 * @param param0
 * @returns
 */
export const AudioMessagingAction: FC<AudioMessagingActionProps> = ({
  className,
  voiceAgent,
  placeholder,
}) => {
  const localMultibandVolume = useMultibandMicrophoneTrackVolume(voiceAgent, 5);

  const { handleDisconnectAgent, handleConnectAgent, isConnected } =
    useConnectAgent(voiceAgent);

  const { handleTextToggle } = useInputModeToggleAgent(voiceAgent);
  const { devices, activeDeviceId, setActiveMediaDevice } =
    useSelectInputDeviceAgent({
      voiceAgent: voiceAgent,
      requestPermissions: true,
    });
  return (
    <div className={cn('relative flex items-center p-2 py-3 gap-4', className)}>
      <div className="flex items-center justify-center w-full">
        {!isConnected ? (
          <button
            onClick={() => {
              handleConnectAgent();
            }}
            className={cn(
              'flex items-center gap-1.5 border-[0.5px] border-primary/10 bg-gray-100 dark:bg-gray-950 rounded-[2px] p-1 shadow-lg  px-4 py-2',
            )}
          >
            <CircleFadingPlus className="w-4 h-4" strokeWidth={1.5} />
            <span className="font-medium text-sm">Click to talk</span>
          </button>
        ) : (
          <div className="flex items-center gap-1.5 border-[0.5px] border-primary/10 bg-gray-100 dark:bg-gray-950 rounded-[2px] p-1 shadow-lg ">
            <div className="rounded-[2px] px-2 py-2 flex items-center gap-2">
              <Mic
                className="w-5 h-5 opacity-50 hover:opacity-100 dark:text-white text-gray-800"
                strokeWidth={1.5}
              />
              <MultibandAudioVisualizerComponent
                classNames="gap-1"
                state="connecting"
                barWidth={3}
                minBarHeight={2}
                maxBarHeight={14}
                barColor="bg-gray-700 dark:bg-white opacity-50 "
                frequencies={
                  localMultibandVolume.length > 0
                    ? localMultibandVolume
                    : Array.from({ length: 3 }, () => [0.01])
                }
              />

              <FlyoutLink
                FlyoutContent={
                  <div className="flex flex-col rounded-[2px] w-[265px] overflow-hidden bg-white dark:bg-gray-950 border-subtle border shadow-lg dark:border-gray-700">
                    <div className="p-1 space-y-1">
                      {devices.map((x, idx) => {
                        return (
                          <div
                            key={idx}
                            className={cn(
                              activeDeviceId === x.deviceId &&
                                'bg-primary! text-white!',
                              'rounded-[2px] px-2.5 py-2.5 hover:bg-primary/10 hover:text-primary hover:rounded-[2px] text-palette-blue-700 text-sm',
                            )}
                          >
                            <div
                              className="flex gap-2 text-xs font-medium items-center"
                              onClick={() => {
                                setActiveMediaDevice(x.deviceId);
                              }}
                            >
                              {x.label}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                }
                className={'flex items-center'}
              >
                {isOpen => (
                  <ChevronDown
                    strokeWidth={1.5}
                    className={cn(
                      'w-5 h-5 transition-transform duration-300 opacity-50 hover:opacity-100 dark:text-white',
                      isOpen ? 'rotate-180' : '',
                    )}
                  />
                )}
              </FlyoutLink>
            </div>
            <Tooltip content="Click for text mode">
              <button
                onClick={async () => {
                  await handleTextToggle();
                }}
                className="bg-primary/20 backdrop-blur-xl rounded-[2px] shadow-lg p-2 border-[0.2px] border-primary/20 text-primary hover:bg-primary hover:text-white"
              >
                <MessageSquareText className="w-4 h-4" strokeWidth="1.5" />
              </button>
            </Tooltip>
            <Tooltip content="Click to end session">
              <button
                onClick={async () => {
                  await handleDisconnectAgent();
                  await handleTextToggle();
                }}
                className="bg-red-500/20 backdrop-blur-xl rounded-[2px] shadow-lg p-2 border-[0.2px] border-red-500/20 text-red-500 hover:bg-red-500 hover:text-white cursor-pointer"
              >
                <X className="w-4 h-4" strokeWidth="1.5" />
              </button>
            </Tooltip>
          </div>
        )}
      </div>
    </div>
  );
};

const FlyoutLink = ({ children, FlyoutContent, className }) => {
  const [open, setOpen] = useState(false);
  return (
    <div
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      className="relative w-fit h-fit"
    >
      <div className={cn(className)}>
        {typeof children === 'function' ? children(open) : children}
      </div>
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: 15 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 15 }}
            transition={{ duration: 0.3, ease: 'easeOut' }}
            className="absolute left-0 bottom-12 z-50"
          >
            <div className="absolute -top-6 left-0 right-0 h-6 bg-transparent" />
            <div className="absolute left-4 -bottom-2 h-4 w-4 rotate-45 bg-white dark:bg-gray-950 border-b border-r dark:border-gray-700" />
            {FlyoutContent}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
