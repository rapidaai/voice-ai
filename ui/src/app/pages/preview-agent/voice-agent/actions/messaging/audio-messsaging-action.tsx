import { FC, HTMLAttributes, useState, useMemo, useCallback } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import {
  Checkmark,
  ChevronDown,
  Microphone,
  MicrophoneOff,
  Chat,
  StopFilledAlt,
} from '@carbon/icons-react';
import { GhostButton } from '@/app/components/ui/primitives';
import {
  useConnectAgent,
  MultibandAudioVisualizerComponent,
  useMultibandMicrophoneTrackVolume,
  useSelectInputDeviceAgent,
  useInputModeToggleAgent,
  useMuteAgent,
  VoiceAgent,
} from '@rapidaai/react';
import { cn } from '@/utils';

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
  // Use higher band count and more responsive settings for better voice visualization
  const localMultibandVolume = useMultibandMicrophoneTrackVolume(
    voiceAgent,
    5, // 5 frequency bands
    0.05, // Lower loPass for more bass frequencies
    0.85, // Higher hiPass for more treble
  );

  const { isConnected, isConnecting, handleDisconnectAgent } =
    useConnectAgent(voiceAgent);
  const { handleTextToggle } = useInputModeToggleAgent(voiceAgent);
  const { isMuted, handleToggleMute } = useMuteAgent(voiceAgent);

  const { devices, activeDeviceId, setActiveMediaDevice } =
    useSelectInputDeviceAgent({
      voiceAgent: voiceAgent,
      requestPermissions: true,
    });

  // Get the currently selected device label
  const activeDeviceLabel = useMemo(() => {
    const device = devices.find(d => d.deviceId === activeDeviceId);
    if (device) {
      // Truncate long device names
      const label = device.label || 'Unknown Device';
      return label.length > 25 ? label.substring(0, 22) + '...' : label;
    }
    return 'Select Microphone';
  }, [devices, activeDeviceId]);

  // Handle smooth device switching
  const handleDeviceChange = useCallback(
    async (deviceId: string) => {
      if (deviceId !== activeDeviceId) {
        await setActiveMediaDevice(deviceId);
      }
    },
    [activeDeviceId, setActiveMediaDevice],
  );

  // Compute visualizer frequencies - when muted, show flat bars
  const visualizerFrequencies = useMemo(() => {
    if (isMuted) {
      // Show minimal flat bars when muted
      return Array.from({ length: 5 }, () => [0.02]);
    }
    return localMultibandVolume.length > 0
      ? localMultibandVolume
      : Array.from({ length: 5 }, () => [0.02]);
  }, [isMuted, localMultibandVolume]);

  return (
    <div className={cn('relative flex items-center p-2 py-3 gap-4', className)}>
      <div className="flex items-center justify-center w-full">
        <div className="flex flex-row border divide-x">
          <button
            type="button"
            disabled={!isConnected}
            onClick={async () => {
              await handleToggleMute();
            }}
            className="group h-10 px-3 flex items-center justify-center border border-support-error text-support-error hover:bg-support-error hover:text-inverse disabled:opacity-50 transition-all duration-200 cursor-pointer"
          >
            {isMuted ? (
              <MicrophoneOff size={16} className="shrink-0" />
            ) : (
              <Microphone size={16} className="shrink-0" />
            )}
            <span className="max-w-0 group-hover:max-w-[120px] overflow-hidden transition-all duration-200 whitespace-nowrap group-hover:ml-2 text-sm font-medium">
              {isMuted ? 'Unmute' : 'Mute'}
            </span>
          </button>
          <div className="px-2 flex items-center gap-2">
            <MultibandAudioVisualizerComponent
              classNames="gap-1"
              state={isMuted ? 'disconnected' : 'listening'}
              barWidth={4}
              minBarHeight={3}
              maxBarHeight={18}
              barColor={cn(
                'rounded-full transition-all duration-150',
                isMuted ? 'bg-support-error/50 ' : 'bg-primary opacity-80',
              )}
              frequencies={visualizerFrequencies}
            />

            {/* Device Selector Dropdown */}
            <FlyoutLink
              FlyoutContent={
                <div className="flex flex-col rounded-[2px] w-[300px] overflow-hidden bg-surface border-subtle border shadow-lg border-border-subtle">
                  <div className="px-3 py-2 border-b border-border-subtle ">
                    <span className="text-xs font-semibold text-muted uppercase tracking-wide">
                      Select Microphone
                    </span>
                  </div>
                  <div className="p-1 space-y-0.5 max-h-[200px] overflow-y-auto">
                    {devices.map((device, idx) => {
                      const isActive = activeDeviceId === device.deviceId;
                      return (
                        <button
                          key={device.deviceId || idx}
                          className={cn(
                            'w-full text-left rounded-[2px] px-3 py-2.5 text-sm transition-all duration-150',
                            'flex items-center justify-between gap-2',
                            isActive
                              ? 'bg-primary text-on-primary'
                              : 'hover:bg-primary/10 text-muted ',
                          )}
                          onClick={() => handleDeviceChange(device.deviceId)}
                        >
                          <div className="flex items-center gap-2 min-w-0 flex-1">
                            <span className="truncate">
                              {device.label || `Microphone ${idx + 1}`}
                            </span>
                          </div>
                          {isActive && (
                            <Checkmark className="w-4 h-4 shrink-0 text-inverse" />
                          )}
                        </button>
                      );
                    })}
                  </div>
                </div>
              }
              className="flex items-center cursor-pointer"
            >
              {isOpen => (
                <div className="flex items-center gap-1 px-2 py-2 rounded-[2px] hover:bg-primary/10 transition-colors">
                  <span className="text-xs font-medium text-muted max-w-[120px] truncate">
                    {activeDeviceLabel}
                  </span>
                  <ChevronDown
                    className={cn(
                      'w-4 h-4 transition-transform duration-300 text-muted ',
                      isOpen ? 'rotate-180' : '',
                    )}
                  />
                </div>
              )}
            </FlyoutLink>
          </div>
          <GhostButton
            size="md"
            disabled={!isConnected}
            renderIcon={Chat}
            onClick={async () => {
              await handleTextToggle();
            }}
          >
            Text
          </GhostButton>
          <button
            type="button"
            disabled={!isConnected && !isConnecting}
            onClick={async () => {
              await handleDisconnectAgent();
            }}
            className="group h-10 px-3 flex items-center justify-center bg-support-error hover:bg-support-error disabled:opacity-50 text-inverse transition-all duration-200 cursor-pointer"
          >
            <StopFilledAlt size={16} className="shrink-0" />
            <span className="max-w-0 group-hover:max-w-[80px] overflow-hidden transition-all duration-200 whitespace-nowrap group-hover:ml-2 text-sm font-medium">
              {isConnecting ? 'Connecting' : 'Stop'}
            </span>
          </button>
        </div>
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
            <div className="absolute left-4 -bottom-2 h-4 w-4 rotate-45 bg-surface border-b border-r border-border-subtle" />
            {FlyoutContent}
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};
