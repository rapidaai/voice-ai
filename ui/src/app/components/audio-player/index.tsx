import { useEffect, useRef, useState, FC, ReactNode, ChangeEvent } from 'react';
import WaveSurfer from 'wavesurfer.js';
import TimelinePlugin from 'wavesurfer.js/dist/plugins/timeline.esm.js';
import { IButton } from '@/app/components/Form/Button';
import { ArrowDownToLine, Pause, Play, Volume2, VolumeX } from 'lucide-react';
import { Tooltip } from '@/app/components/base/tooltip';
import { cn } from '@/styles/media';
import { Slider } from '@/app/components/Form/Slider';

type AudioPlayerProps = {
  src: string;
  progressColor?: string;
  cursorColor?: string;
  buttonsColor?: string;
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

export const AudioPlayer: FC<AudioPlayerProps> = ({
  src,
  progressColor = 'blue',
  cursorColor = 'blue',
  barWidth = 2,
  barRadius = 2,
  barGap = 1,
  height = 100,
  playbackSpeeds = [1, 1.5, 2],
  onPlay,
  onPause,
  onVolumeChange,
}) => {
  const waveformRef = useRef<HTMLDivElement | null>(null);
  const wavesurfer = useRef<WaveSurfer | null>(null);

  const [playing, setPlaying] = useState<boolean>(false);
  const [volume, setVolume] = useState<number>(1);
  const [muted, setMuted] = useState<boolean>(false);
  const [currentTime, setCurrentTime] = useState<string>('0:00');
  const [duration, setDuration] = useState<string>('0:00');
  const [playBackSpeed, setPlayBackSpeed] = useState(playbackSpeeds[0]);

  useEffect(() => {
    if (waveformRef.current) {
      wavesurfer.current = WaveSurfer.create({
        container: waveformRef.current,
        // waveColor,
        progressColor,
        cursorColor,
        barWidth,
        barGap,
        barRadius,
        height,
        normalize: true,
        audioRate: playBackSpeed,
        plugins: [
          TimelinePlugin.create({
            timeInterval: 1,
          }),
        ],
      });

      wavesurfer.current.load(src);

      wavesurfer.current.on('ready', () => {
        setDuration(formatTime(wavesurfer.current?.getDuration() || 0));
      });

      wavesurfer.current.on('audioprocess', () => {
        setCurrentTime(formatTime(wavesurfer.current?.getCurrentTime() || 0));
      });
    }

    return () => {
      wavesurfer.current?.destroy();
    };
  }, [src, progressColor, cursorColor, barWidth, barRadius, barGap, height]);

  useEffect(() => {
    if (wavesurfer.current) {
      wavesurfer.current.setPlaybackRate(playBackSpeed);
    }
  }, [playBackSpeed]);

  const togglePlay = () => {
    if (wavesurfer.current) {
      wavesurfer.current.playPause();
      setPlaying(!playing);
      if (playing) {
        onPause?.();
      } else {
        onPlay?.();
      }
    }
  };

  const handleVolume = (newVolume: number) => {
    // const newVolume = parseFloat(e);
    if (wavesurfer.current) {
      if (muted) {
        setMuted(false);
      }
      wavesurfer.current.setVolume(newVolume);
      setVolume(newVolume);
      onVolumeChange?.(newVolume);
    }
  };

  const toggleMute = () => {
    if (wavesurfer.current) {
      wavesurfer.current.setVolume(muted ? volume : 0);
      setMuted(!muted);
    }
  };

  const formatTime = (seconds: number) => {
    const minutes = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${minutes}:${secs < 10 ? '0' : ''}${secs}`;
  };

  const handleDownloadAudio = async () => {
    try {
      const response = await fetch(src);
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'conversation-complete-recording.wav';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.log(error);
    }
  };

  return (
    <div className={`flex w-full flex-col items-center rounded-lg`}>
      <div ref={waveformRef} className="w-full z-0" />
      <div className="flex w-full flex-col justify-between gap-3 md:flex-row md:items-center bg-white dark:bg-gray-900 border-y">
        <div className="flex items-center justify-between divide-x border-r">
          <IButton type="button" onClick={togglePlay}>
            {playing ? (
              <Pause className="w-4 h-4" strokeWidth={1.5} />
            ) : (
              <Play className="w-4 h-4" strokeWidth={1.5} />
            )}
          </IButton>
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
            <IButton onClick={toggleMute} type="button">
              {muted || volume === 0 ? (
                <VolumeX className="w-4 h-4" strokeWidth={1.5} />
              ) : (
                <Volume2 className="w-4 h-4" strokeWidth={1.5} />
              )}
            </IButton>
          </Tooltip>
        </div>
        <div className="flex items-center justify-between divide-x border-l">
          {playbackSpeeds.map(speed => (
            <IButton
              key={speed}
              onClick={() => setPlayBackSpeed(speed)}
              className={cn(
                speed === playBackSpeed &&
                  'bg-blue-600 text-white hover:bg-blue-600!',
              )}
            >
              {speed}x
            </IButton>
          ))}
          <IButton
            onClick={handleDownloadAudio}
            type="button"
            className="border-x"
          >
            <ArrowDownToLine className="h-4 w-4 mr-1" /> <span>Audio</span>
          </IButton>
        </div>
      </div>
    </div>
  );
};
