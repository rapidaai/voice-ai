import React from 'react';
import { act, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { setRangeValue } from '@/test-utils/set-range-value';
import '@testing-library/jest-dom';

import { AudioPlayer } from '../audio-player';

const mockCreate = jest.fn();
const mockTimelineCreate = jest.fn(() => ({ name: 'timeline' }));

jest.mock('wavesurfer.js', () => ({
  __esModule: true,
  default: {
    create: (...args: unknown[]) => mockCreate.apply(null, args),
  },
}));

jest.mock('wavesurfer.js/dist/plugins/timeline.esm.js', () => ({
  __esModule: true,
  default: {
    create: (...args: unknown[]) => mockTimelineCreate.apply(null, args),
  },
}));

jest.mock('@carbon/icons-react', () => ({
  Download: () => <svg data-testid="download-icon" />,
  Pause: () => <svg data-testid="pause-icon" />,
  Play: () => <svg data-testid="play-icon" />,
  VolumeMute: () => <svg data-testid="volume-mute-icon" />,
  VolumeUp: () => <svg data-testid="volume-up-icon" />,
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  GhostButton: ({
    children,
    disabled,
    onClick,
    type = 'button',
  }: React.PropsWithChildren<any>) => (
    <button disabled={disabled} onClick={onClick} type={type}>
      {children}
    </button>
  ),
}));

jest.mock('@/app/components/ui/primitives/floating-tooltip', () => ({
  Tooltip: ({ children, content }: React.PropsWithChildren<any>) => (
    <div>
      {children}
      {content}
    </div>
  ),
}));

jest.mock('@/app/components/ui/primitives/slider', () => ({
  Slider: ({ onSlide, value }: any) => (
    <input
      aria-label="Volume"
      type="range"
      value={value}
      onChange={event => onSlide(Number(event.target.value))}
    />
  ),
}));

type WaveSurferHandlers = Record<string, (...args: any[]) => void>;

const createWaveSurfer = () => {
  const handlers: WaveSurferHandlers = {};
  return {
    handlers,
    destroy: jest.fn(),
    getCurrentTime: jest.fn(() => 12),
    getDuration: jest.fn(() => 120),
    load: jest.fn(),
    on: jest.fn((event: string, handler: (...args: any[]) => void) => {
      handlers[event] = handler;
    }),
    pause: jest.fn(),
    play: jest.fn(() => Promise.resolve()),
    seekTo: jest.fn(),
    setPlaybackRate: jest.fn(),
    setVolume: jest.fn(),
  };
};

const recording = {
  getAssistantrecordingurl: () => '/assistant.wav',
  getConversationrecordingurl: () => '/conversation.wav',
  getId: () => 'recording-1',
  getUserrecordingurl: () => '/user.wav',
} as any;

describe('AudioPlayer', () => {
  const audioContext = {
    close: jest.fn(),
    resume: jest.fn(),
    state: 'running',
  };

  beforeEach(() => {
    jest.clearAllMocks();
    (global as any).AudioContext = jest.fn(() => audioContext);
  });

  it('loads both recording tracks and disables playback until ready', () => {
    const assistantWave = createWaveSurfer();
    const userWave = createWaveSurfer();
    mockCreate.mockReturnValueOnce(assistantWave).mockReturnValueOnce(userWave);

    render(<AudioPlayer recording={recording} />);

    expect(mockCreate).toHaveBeenCalledTimes(2);
    expect(assistantWave.load).toHaveBeenCalledWith('/assistant.wav');
    expect(userWave.load).toHaveBeenCalledWith('/user.wav');
    expect(screen.getByTestId('play-icon').closest('button')).toBeDisabled();
  });

  it('plays and pauses synchronized tracks once both are ready', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onPlay = jest.fn();
    const onPause = jest.fn();
    const assistantWave = createWaveSurfer();
    const userWave = createWaveSurfer();
    mockCreate.mockReturnValueOnce(assistantWave).mockReturnValueOnce(userWave);

    render(
      <AudioPlayer recording={recording} onPlay={onPlay} onPause={onPause} />,
    );

    act(() => {
      assistantWave.handlers.ready();
      userWave.handlers.ready();
    });

    const playButton = screen.getByTestId('play-icon').closest('button')!;
    await act(async () => {
      await user.click(playButton);
    });

    expect(userWave.seekTo).toHaveBeenCalledWith(0.1);
    expect(assistantWave.play).toHaveBeenCalledTimes(1);
    expect(userWave.play).toHaveBeenCalledTimes(1);
    expect(onPlay).toHaveBeenCalledTimes(1);

    await act(async () => {
      await user.click(screen.getByTestId('pause-icon').closest('button')!);
    });

    expect(assistantWave.pause).toHaveBeenCalled();
    expect(userWave.pause).toHaveBeenCalled();
    expect(onPause).toHaveBeenCalledTimes(1);
  });

  it('updates volume and downloads the selected recording', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const onVolumeChange = jest.fn();
    const assistantWave = createWaveSurfer();
    const userWave = createWaveSurfer();
    const appendChild = jest.spyOn(document.body, 'appendChild');
    const removeChild = jest.spyOn(document.body, 'removeChild');
    const click = jest.spyOn(HTMLAnchorElement.prototype, 'click');

    mockCreate.mockReturnValueOnce(assistantWave).mockReturnValueOnce(userWave);
    global.fetch = jest.fn(() =>
      Promise.resolve({ blob: () => Promise.resolve(new Blob(['audio'])) }),
    ) as any;
    global.URL.createObjectURL = jest.fn(() => 'blob:audio');
    global.URL.revokeObjectURL = jest.fn();

    render(
      <AudioPlayer recording={recording} onVolumeChange={onVolumeChange} />,
    );

    act(() => {
      assistantWave.handlers.ready();
      userWave.handlers.ready();
    });

    const volume = screen.getByRole('slider', { name: 'Volume' });
    await user.click(volume);
    setRangeValue(volume as HTMLInputElement, 0.5);
    await user.click(screen.getByRole('button', { name: /conversation/i }));

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith('/conversation.wav');
    });
    expect(assistantWave.setVolume).toHaveBeenCalledWith(0.5);
    expect(userWave.setVolume).toHaveBeenCalledWith(0.5);
    expect(onVolumeChange).toHaveBeenCalledWith(0.5);
    expect(appendChild).toHaveBeenCalled();
    expect(click).toHaveBeenCalled();
    expect(removeChild).toHaveBeenCalled();
    expect(global.URL.revokeObjectURL).toHaveBeenCalledWith('blob:audio');
  });
});
