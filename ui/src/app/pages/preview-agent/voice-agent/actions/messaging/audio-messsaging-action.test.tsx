import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';

import { AudioMessagingAction } from './audio-messsaging-action';

const mockHandleDisconnectAgent = jest.fn();
const mockHandleTextToggle = jest.fn();
const mockHandleToggleMute = jest.fn();
const mockSetActiveMediaDevice = jest.fn();

let mockActiveDeviceId = 'mic-1';

jest.mock('framer-motion', () => ({
  AnimatePresence: ({ children }: { children: React.ReactNode }) => (
    <>{children}</>
  ),
  motion: {
    div: ({ children, ...props }: React.ComponentProps<'div'>) => (
      <div {...props}>{children}</div>
    ),
  },
}));

jest.mock('@rapidaai/react', () => ({
  MultibandAudioVisualizerComponent: () => (
    <div data-testid="audio-visualizer" />
  ),
  useConnectAgent: () => ({
    handleDisconnectAgent: mockHandleDisconnectAgent,
    isConnected: true,
    isConnecting: false,
  }),
  useInputModeToggleAgent: () => ({
    handleTextToggle: mockHandleTextToggle,
  }),
  useMultibandMicrophoneTrackVolume: () => [[0.2], [0.3], [0.4]],
  useMuteAgent: () => ({
    handleToggleMute: mockHandleToggleMute,
    isMuted: false,
  }),
  useSelectInputDeviceAgent: () => ({
    activeDeviceId: mockActiveDeviceId,
    devices: [
      { deviceId: 'mic-1', label: 'Built-in Microphone' },
      { deviceId: 'mic-2', label: 'USB Microphone' },
    ],
    setActiveMediaDevice: mockSetActiveMediaDevice,
  }),
}));

jest.mock('@carbon/icons-react', () => ({
  Chat: ({ className }: { className?: string }) => (
    <svg className={className} data-testid="chat-icon" />
  ),
  Checkmark: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="checkmark-icon"
    />
  ),
  ChevronDown: ({ className, strokeWidth }: any) => (
    <svg
      className={className}
      data-stroke-width={strokeWidth}
      data-testid="chevron-down-icon"
    />
  ),
  Microphone: ({ className }: { className?: string }) => (
    <svg className={className} data-testid="microphone-icon" />
  ),
  MicrophoneOff: ({ className }: { className?: string }) => (
    <svg className={className} data-testid="microphone-off-icon" />
  ),
  StopFilledAlt: ({ className }: { className?: string }) => (
    <svg className={className} data-testid="stop-icon" />
  ),
}));

jest.mock('@/app/components/ui/primitives/button', () => ({
  GhostButton: ({
    children,
    renderIcon: Icon,
    ...props
  }: React.ComponentProps<'button'> & { renderIcon?: React.ElementType }) => (
    <button type="button" {...props}>
      {Icon ? <Icon /> : null}
      {children}
    </button>
  ),
}));

describe('AudioMessagingAction', () => {
  beforeEach(() => {
    mockActiveDeviceId = 'mic-1';
    jest.clearAllMocks();
  });

  it('renders microphone controls with Carbon dropdown icons', () => {
    render(<AudioMessagingAction voiceAgent={{} as any} />);

    expect(screen.getByText('Built-in Microphone')).toBeInTheDocument();
    expect(screen.getByTestId('microphone-icon')).toBeInTheDocument();
    expect(screen.getByTestId('chevron-down-icon')).not.toHaveAttribute(
      'data-stroke-width',
    );
  });

  it('shows the active microphone with a Carbon checkmark and switches devices', async () => {
    const user = userEvent.setup({ skipHover: true, writeToClipboard: false });
    render(<AudioMessagingAction voiceAgent={{} as any} />);

    await user.hover(screen.getByText('Built-in Microphone'));

    const activeIcons = await screen.findAllByTestId('checkmark-icon');
    expect(activeIcons).toHaveLength(1);
    expect(activeIcons[0]).not.toHaveAttribute('data-stroke-width');

    await user.click(screen.getByRole('button', { name: 'USB Microphone' }));

    await waitFor(() =>
      expect(mockSetActiveMediaDevice).toHaveBeenCalledWith('mic-2'),
    );
  });

  it('uses the connected action handlers', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(<AudioMessagingAction voiceAgent={{} as any} />);

    await user.click(screen.getByRole('button', { name: /mute/i }));
    await user.click(screen.getByRole('button', { name: /text/i }));
    await user.click(screen.getByRole('button', { name: /stop/i }));

    expect(mockHandleToggleMute).toHaveBeenCalledTimes(1);
    expect(mockHandleTextToggle).toHaveBeenCalledTimes(1);
    expect(mockHandleDisconnectAgent).toHaveBeenCalledTimes(1);
  });
});
