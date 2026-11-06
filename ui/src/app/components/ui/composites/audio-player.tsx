import type { ReactNode } from 'react';
import { useEffect, useRef, useState } from 'react';
import WaveSurfer from 'wavesurfer.js';
import TimelinePlugin from 'wavesurfer.js/dist/plugins/timeline.esm.js';
import { GhostButton } from '@/app/components/ui/primitives/button';
import {
  Download,
  Pause,
  Play,
  VolumeMute,
  VolumeUp,
} from '@carbon/icons-react';
import { Tooltip } from '@/app/components/ui/primitives/floating-tooltip';
import { cn } from '@/utils';
import { Slider } from '@/app/components/ui/primitives/slider';
import { AssistantConversationRecording } from '@rapidaai/react';

type AudioPlayerProps = {
  recording: AssistantConversationRecording;
  barWidth?: number;
  barRadius?: number;
  barGap?: number;
  height?: number;
  volumeUpIcon?: ReactNode;
  volumeMuteIcon?: ReactNode;
  playbackSpeeds?: number[];
  onPlay?: () => void;
  onPause?: () => void;
  onVolumeChange?: (volume: number) => void;
};

export function AudioPlayer({
  recording,
  barWidth = 2,
  barRadius = 2,
  barGap = 1,
  height = 100,
  volumeUpIcon,
  volumeMuteIcon,
  playbackSpeeds = [1, 1.5, 2],
  onPlay,
  onPause,
  onVolumeChange,
}: AudioPlayerProps) {
  const assistantProgressColor = 'var(--color-data-2)';
  const userProgressColor = 'var(--color-data-4)';
  const cursorColor = 'var(--color-data-2)';
  const assistantWaveformRef = useRef<HTMLDivElement | null>(null);
  const userWaveformRef = useRef<HTMLDivElement | null>(null);
  const timelineRef = useRef<HTMLDivElement | null>(null);
  const assistantWavesurfer = useRef<WaveSurfer | null>(null);
  const userWavesurfer = useRef<WaveSurfer | null>(null);
  const sharedAudioContext = useRef<AudioContext | null>(null);

  const [playing, setPlaying] = useState<boolean>(false);
  const [volume, setVolume] = useState<number>(1);
  const [muted, setMuted] = useState<boolean>(false);
  const [, setCurrentTime] = useState<string>('0:00');
  const [, setDuration] = useState<string>('0:00');
  const [playBackSpeed, setPlayBackSpeed] = useState(playbackSpeeds[0]);
  const [bothReady, setBothReady] = useState<boolean>(false);
  const readyCount = useRef<number>(0);

  const assistantSrc = recording.getAssistantrecordingurl();
  const userSrc = recording.getUserrecordingurl();
  const completeRecordingSrc = recording.getConversationrecordingurl();
  const renderVolumeIcon =
    muted || volume === 0
      ? volumeMuteIcon || <VolumeMute className="w-4 h-4" strokeWidth={1.5} />
      : volumeUpIcon || <VolumeUp className="w-4 h-4" strokeWidth={1.5} />;

  useEffect(() => {
    if (!sharedAudioContext.current) {
      sharedAudioContext.current = new AudioContext();
    }
    return () => {
      sharedAudioContext.current?.close();
      sharedAudioContext.current = null;
    };
  }, []);

  const createWaveSurferOptions = (
    container: HTMLDivElement,
    progressColor: string,
  ) => ({
    container,
    progressColor,
    cursorColor,
    barWidth,
    barGap,
    barRadius,
    height,
    normalize: true,
    audioRate: playBackSpeed,
    backend: 'WebAudio' as const,
    audioContext: sharedAudioContext.current!,
    plugins: [],
  });

  useEffect(() => {
    readyCount.current = 0;
    setBothReady(false);

    const onTrackReady = () => {
      readyCount.current += 1;
      if (readyCount.current >= 2) {
        setBothReady(true);
      }
    };

    if (assistantWaveformRef.current) {
      assistantWavesurfer.current = WaveSurfer.create({
        ...createWaveSurferOptions(
          assistantWaveformRef.current,
          assistantProgressColor,
        ),
        ...(timelineRef.current
          ? {
              plugins: [
                TimelinePlugin.create({
                  timeInterval: 1,
                  container: timelineRef.current,
                }),
              ],
            }
          : {}),
      });
      assistantWavesurfer.current.load(assistantSrc);

      assistantWavesurfer.current.on('ready', () => {
        setDuration(
          formatTime(assistantWavesurfer.current?.getDuration() || 0),
        );
        onTrackReady();
      });

      assistantWavesurfer.current.on('audioprocess', () => {
        setCurrentTime(
          formatTime(assistantWavesurfer.current?.getCurrentTime() || 0),
        );
      });

      assistantWavesurfer.current.on('seeking', (currentTime: number) => {
        if (userWavesurfer.current) {
          const userDuration = userWavesurfer.current.getDuration();
          if (userDuration > 0) {
            userWavesurfer.current.seekTo(currentTime / userDuration);
          }
        }
      });

      assistantWavesurfer.current.on('finish', () => {
        userWavesurfer.current?.pause();
        setPlaying(false);
      });
    }

    if (userWaveformRef.current) {
      userWavesurfer.current = WaveSurfer.create(
        createWaveSurferOptions(userWaveformRef.current, userProgressColor),
      );
      userWavesurfer.current.load(userSrc);

      userWavesurfer.current.on('ready', () => {
        onTrackReady();
      });

      userWavesurfer.current.on('finish', () => {
        assistantWavesurfer.current?.pause();
        setPlaying(false);
      });
    }

    return () => {
      try {
        assistantWavesurfer.current?.pause();
      } catch {}
      try {
        userWavesurfer.current?.pause();
      } catch {}
      try {
        assistantWavesurfer.current?.destroy();
      } catch {}
      try {
        userWavesurfer.current?.destroy();
      } catch {}
      assistantWavesurfer.current = null;
      userWavesurfer.current = null;
    };
  }, [
    assistantSrc,
    userSrc,
    assistantProgressColor,
    userProgressColor,
    cursorColor,
    barWidth,
    barRadius,
    barGap,
    height,
  ]);

  useEffect(() => {
    assistantWavesurfer.current?.setPlaybackRate(playBackSpeed);
    userWavesurfer.current?.setPlaybackRate(playBackSpeed);
  }, [playBackSpeed]);

  const togglePlay = async () => {
    if (!bothReady || !assistantWavesurfer.current || !userWavesurfer.current) {
      return;
    }

    if (sharedAudioContext.current?.state === 'suspended') {
      await sharedAudioContext.current.resume();
    }

    if (playing) {
      assistantWavesurfer.current.pause();
      userWavesurfer.current.pause();
      onPause?.();
    } else {
      const currentTime = assistantWavesurfer.current.getCurrentTime();
      const userDuration = userWavesurfer.current.getDuration();
      if (userDuration > 0) {
        userWavesurfer.current.seekTo(currentTime / userDuration);
      }

      await Promise.all([
        assistantWavesurfer.current.play(),
        userWavesurfer.current.play(),
      ]);
      onPlay?.();
    }
    setPlaying(!playing);
  };

  const handleVolume = (newVolume: number) => {
    if (assistantWavesurfer.current && userWavesurfer.current) {
      if (muted) setMuted(false);
      assistantWavesurfer.current.setVolume(newVolume);
      userWavesurfer.current.setVolume(newVolume);
      setVolume(newVolume);
      onVolumeChange?.(newVolume);
    }
  };

  const toggleMute = () => {
    if (assistantWavesurfer.current && userWavesurfer.current) {
      const newVolume = muted ? volume : 0;
      assistantWavesurfer.current.setVolume(newVolume);
      userWavesurfer.current.setVolume(newVolume);
      setMuted(!muted);
    }
  };

  const formatTime = (seconds: number) => {
    const minutes = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${minutes}:${secs < 10 ? '0' : ''}${secs}`;
  };

  const handleDownloadAudio = async (
    recordingId: string,
    src: string,
    label: string,
  ) => {
    try {
      const response = await fetch(src);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${label}-${recordingId}.wav`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.log(error);
    }
  };

  return (
    <div className="flex w-full flex-col items-center rounded-lg">
      <div className="relative w-full" style={{ height }}>
        <div
          ref={userWaveformRef}
          className="absolute inset-0 z-[1] pointer-events-none"
          style={{ opacity: 0.5 }}
        />
        <div
          ref={assistantWaveformRef}
          className="absolute inset-0 z-[2]"
          style={{ opacity: 0.7 }}
        />
      </div>

      <div ref={timelineRef} className="w-full" />

      <div className="flex w-full flex-col justify-between gap-3 border-y bg-surface md:flex-row md:items-center">
        <div className="flex items-center justify-between divide-x border-r">
          <GhostButton
            size="md"
            type="button"
            onClick={togglePlay}
            disabled={!bothReady}
          >
            {playing ? (
              <Pause className="w-4 h-4" strokeWidth={1.5} />
            ) : (
              <Play className="w-4 h-4" strokeWidth={1.5} />
            )}
          </GhostButton>

          <Tooltip
            content={
              <Slider
                type="range"
                min="0"
                max="1"
                step="0.01"
                value={muted ? 0 : volume}
                onSlide={x => {
                  handleVolume(x);
                }}
              />
            }
          >
            <GhostButton size="md" onClick={toggleMute} type="button">
              {renderVolumeIcon}
            </GhostButton>
          </Tooltip>
        </div>

        <div className="flex items-center justify-between divide-x border-l">
          {playbackSpeeds.map(speed => (
            <GhostButton
              size="md"
              key={speed}
              onClick={() => setPlayBackSpeed(speed)}
              className={cn(
                'rounded-none',
                speed === playBackSpeed &&
                  'bg-support-info text-inverse hover:bg-support-info!',
              )}
            >
              {speed}x
            </GhostButton>
          ))}
          <GhostButton
            size="md"
            onClick={() =>
              handleDownloadAudio(
                recording.getId(),
                completeRecordingSrc,
                'conversation',
              )
            }
            type="button"
            className="rounded-none"
          >
            <Download
              className="mr-2 stroke-primary size-4"
              strokeWidth={0.2}
            />{' '}
            <span>Conversation</span>
          </GhostButton>
          <GhostButton
            size="md"
            onClick={() =>
              handleDownloadAudio(recording.getId(), assistantSrc, 'assistant')
            }
            type="button"
            className="rounded-none"
          >
            <Download
              className="mr-2 stroke-primary size-4"
              strokeWidth={0.2}
            />{' '}
            <span>Assistant</span>
          </GhostButton>
          <GhostButton
            size="md"
            onClick={() =>
              handleDownloadAudio(recording.getId(), userSrc, 'user')
            }
            type="button"
            className="rounded-none"
          >
            <Download
              className="mr-2 stroke-primary size-4"
              strokeWidth={0.2}
            />{' '}
            <span>User</span>
          </GhostButton>
        </div>
      </div>
    </div>
  );
}
