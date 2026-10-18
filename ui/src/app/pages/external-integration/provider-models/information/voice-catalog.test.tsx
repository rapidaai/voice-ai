import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { MemoryRouter } from 'react-router-dom';

import { VoiceCatalog, VoiceCatalogItem } from './voice-catalog';

jest.mock('@/app/components/ui/primitives/pagination', () => ({
  Pagination: ({ onChange, page, pageSize, pageSizes, totalItems }: any) => (
    <button
      type="button"
      data-page-size={pageSize}
      data-page-sizes={pageSizes.join(',')}
      onClick={() => onChange({ page: page + 1, pageSize })}
    >
      Next page ({totalItems})
    </button>
  ),
}));

jest.mock('@/app/components/ui/feedback', () => ({
  EmptyState: ({ title, subtitle }: any) => (
    <section>
      <h2>{title}</h2>
      <p>{subtitle}</p>
    </section>
  ),
}));

const voices: VoiceCatalogItem[] = Array.from({ length: 15 }, (_, index) => ({
  title: `Voice ${index + 1}`,
  voiceId: `voice-${index + 1}`,
  languages: index === 14 ? ['Spanish'] : ['English'],
  persona: [],
  features: index === 14 ? ['Sales'] : ['Support'],
  previewUrl: `https://example.test/voice-${index + 1}.mp3`,
}));

describe('VoiceCatalog', () => {
  const audioInstances: Array<{
    pause: jest.Mock;
    play: jest.Mock;
    onended: null | (() => void);
    onerror: null | (() => void);
  }> = [];
  const OriginalAudio = window.Audio;

  beforeEach(() => {
    audioInstances.length = 0;
    window.Audio = jest.fn().mockImplementation(() => {
      const audio = {
        pause: jest.fn(),
        play: jest.fn().mockResolvedValue(undefined),
        onended: null,
        onerror: null,
      };
      audioInstances.push(audio);
      return audio;
    }) as any;
  });

  afterEach(() => {
    window.Audio = OriginalAudio;
    jest.clearAllMocks();
  });

  it('uses the platform page size and lets users reach remaining voices', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <MemoryRouter>
        <VoiceCatalog voices={voices} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId('voice-catalog-table')).toBeInTheDocument();
    expect(screen.getAllByRole('row')).toHaveLength(11);
    expect(screen.queryByText('Voice 15')).not.toBeInTheDocument();

    const pagination = screen.getByRole('button', { name: 'Next page (15)' });
    expect(pagination).toHaveAttribute('data-page-size', '10');
    expect(pagination).toHaveAttribute('data-page-sizes', '10,20,50,100');

    await user.click(pagination);

    expect(screen.getAllByRole('row')).toHaveLength(6);
    expect(screen.getByText('Voice 15')).toBeInTheDocument();
  });

  it('keeps every voice field visible and expands the description', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <MemoryRouter>
        <VoiceCatalog
          voices={[
            {
              ...voices[0],
              description: 'A clear voice for support conversations.',
              persona: ['Warm'],
              features: ['Support'],
            },
          ]}
        />
      </MemoryRouter>,
    );

    expect(
      screen.queryByText('A clear voice for support conversations.'),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: 'Voice' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: 'Languages' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: 'Persona' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: 'Use cases' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: 'Voice ID' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: 'Preview' }),
    ).toBeInTheDocument();
    expect(screen.getByText('Warm')).toBeInTheDocument();
    expect(screen.getByText('Support')).toBeInTheDocument();
    expect(screen.getByText('voice-1')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Copy Voice 1 voice ID' }),
    ).toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: 'Expand Voice 1 details' }),
    );

    expect(
      screen.getByText('A clear voice for support conversations.'),
    ).toBeInTheDocument();
  });

  it('keeps the preview column visible when previews are unavailable', () => {
    render(
      <MemoryRouter>
        <VoiceCatalog
          voices={voices.slice(0, 2).map(voice => ({
            ...voice,
            previewUrl: undefined,
          }))}
        />
      </MemoryRouter>,
    );

    expect(
      screen.getByRole('columnheader', { name: 'Preview' }),
    ).toBeInTheDocument();
    expect(screen.getAllByLabelText('Preview unavailable')).toHaveLength(2);
  });

  it('uses one Carbon toolbar for expandable search and page actions', async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <VoiceCatalog
          voices={voices}
          actions={<button type="button">Add new credential</button>}
        />
      </MemoryRouter>,
    );

    const search = screen.getByRole('searchbox', { name: 'Search voices' });
    const action = screen.getByRole('button', { name: 'Add new credential' });
    const searchContainer = search.closest(
      '.cds--toolbar-search-container-expandable',
    );
    expect(search.closest('.cds--table-toolbar')).toBe(
      action.closest('.cds--table-toolbar'),
    );
    expect(
      screen.queryByRole('heading', { name: 'Voice catalogue' }),
    ).not.toBeInTheDocument();

    expect(searchContainer).toHaveClass(
      'cds--toolbar-search-container-expandable',
    );
    expect(searchContainer).not.toHaveClass(
      'cds--toolbar-search-container-active',
    );

    await user.click(search);
    expect(searchContainer).toHaveClass('cds--toolbar-search-container-active');

    await user.tab();
    expect(searchContainer).not.toHaveClass(
      'cds--toolbar-search-container-active',
    );
  });

  it('filters voice metadata from the URL and reports an empty result', async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={['/integration/models/test?query=spanish']}>
        <VoiceCatalog voices={voices} />
      </MemoryRouter>,
    );

    expect(screen.getByText('Voice 15')).toBeInTheDocument();
    expect(screen.queryByText('Voice 1')).not.toBeInTheDocument();

    await user.click(screen.getByRole('searchbox', { name: 'Search voices' }));
    await user.keyboard('{Control>}a{/Control}');
    await user.paste(String('not-found'));

    expect(screen.getByText('No voices found')).toBeInTheDocument();
  });

  it('creates audio only on demand and stops the previous preview', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    render(
      <MemoryRouter>
        <VoiceCatalog voices={voices.slice(0, 2)} />
      </MemoryRouter>,
    );

    expect(window.Audio).not.toHaveBeenCalled();

    await user.click(screen.getByRole('button', { name: 'Preview Voice 1' }));
    expect(window.Audio).toHaveBeenCalledTimes(1);
    expect(audioInstances[0].play).toHaveBeenCalledTimes(1);

    await user.click(screen.getByRole('button', { name: 'Preview Voice 2' }));
    expect(audioInstances[0].pause).toHaveBeenCalledTimes(1);
    expect(window.Audio).toHaveBeenCalledTimes(2);
    expect(audioInstances[1].play).toHaveBeenCalledTimes(1);
  });

  it('copies a voice ID from the Carbon table action', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    const writeText = jest.spyOn(navigator.clipboard, 'writeText');
    render(
      <MemoryRouter>
        <VoiceCatalog voices={voices.slice(0, 1)} />
      </MemoryRouter>,
    );

    await user.click(
      screen.getByRole('button', { name: 'Copy Voice 1 voice ID' }),
    );

    expect(writeText).toHaveBeenCalledWith('voice-1');
  });

  it('ignores a stale failure after a newer preview starts', async () => {
    const user = userEvent.setup({ writeToClipboard: false });
    let rejectFirstPreview: (() => void) | undefined;
    (window.Audio as unknown as jest.Mock)
      .mockImplementationOnce(() => {
        const audio = {
          pause: jest.fn(),
          play: jest.fn(
            () =>
              new Promise<void>((_resolve, reject) => {
                rejectFirstPreview = () => reject(new Error('stale failure'));
              }),
          ),
          onended: null,
          onerror: null,
        };
        audioInstances.push(audio);
        return audio;
      })
      .mockImplementationOnce(() => {
        const audio = {
          pause: jest.fn(),
          play: jest.fn().mockResolvedValue(undefined),
          onended: null,
          onerror: null,
        };
        audioInstances.push(audio);
        return audio;
      });

    render(
      <MemoryRouter>
        <VoiceCatalog voices={voices.slice(0, 2)} />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole('button', { name: 'Preview Voice 1' }));
    await user.click(screen.getByRole('button', { name: 'Preview Voice 2' }));
    rejectFirstPreview?.();
    await Promise.resolve();

    expect(audioInstances[1].pause).not.toHaveBeenCalled();
    expect(
      screen.getByRole('button', { name: 'Pause Voice 2' }),
    ).toBeInTheDocument();
  });
});
