import { EmptyState } from '@/app/components/ui/feedback';
import { Pagination } from '@/app/components/ui/primitives';
import { Pause, Play, Search } from '@carbon/icons-react';
import {
  Button,
  CopyButton,
  DataTable,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableExpandedRow,
  TableExpandHeader,
  TableExpandRow,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
  TableToolbarSearch,
  Tag,
} from '@carbon/react';
import {
  ChangeEvent,
  ReactNode,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useLocation } from 'react-router-dom';

const DEFAULT_PAGE_SIZE = 10;
const PAGE_SIZES = [10, 20, 50, 100];
const VOICE_TABLE_HEADERS = [
  { key: 'name', header: 'Voice' },
  { key: 'languages', header: 'Languages' },
  { key: 'persona', header: 'Persona' },
  { key: 'features', header: 'Use cases' },
  { key: 'voiceId', header: 'Voice ID' },
  { key: 'preview', header: 'Preview' },
];
const VOICE_TABLE_COLUMN_WIDTHS = [
  '5%',
  '18%',
  '11%',
  '14%',
  '20%',
  '23%',
  '9%',
];

export interface VoiceCatalogItem {
  title: string;
  voiceId: string;
  description?: string | null;
  previewUrl?: string;
  languages: Array<string | undefined>;
  persona: Array<string | undefined>;
  features: Array<string | undefined>;
}

function availableValues(values: Array<string | undefined>) {
  return values.filter((value): value is string => Boolean(value));
}

function MetadataTags({ values }: { values: Array<string | undefined> }) {
  const visibleValues = availableValues(values);

  if (visibleValues.length === 0) {
    return <span className="text-sm text-muted">—</span>;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {visibleValues.map(value => (
        <Tag key={value} size="sm" type="gray">
          {value}
        </Tag>
      ))}
    </div>
  );
}

export function VoiceCatalog(props: {
  voices: VoiceCatalogItem[];
  actions?: ReactNode;
}) {
  const { actions, voices } = props;
  const location = useLocation();
  const queryFromUrl = useMemo(
    () => new URLSearchParams(location.search).get('query') ?? '',
    [location.search],
  );
  const [query, setQuery] = useState(queryFromUrl);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [playingVoiceId, setPlayingVoiceId] = useState<string | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    setQuery(queryFromUrl);
    setPage(1);
  }, [queryFromUrl]);

  useEffect(
    () => () => {
      const audio = audioRef.current;
      if (!audio) return;
      audio.onended = null;
      audio.onerror = null;
      audio.pause();
      audioRef.current = null;
    },
    [],
  );

  const filteredVoices = useMemo(() => {
    const searchTerm = query.trim().toLocaleLowerCase();
    if (!searchTerm) return voices;

    return voices.filter(voice =>
      [
        voice.title,
        voice.voiceId,
        voice.description,
        ...voice.languages,
        ...voice.persona,
        ...voice.features,
      ]
        .filter(Boolean)
        .join(' ')
        .toLocaleLowerCase()
        .includes(searchTerm),
    );
  }, [query, voices]);

  const firstItemIndex = (page - 1) * pageSize;
  const visibleVoices = filteredVoices.slice(
    firstItemIndex,
    firstItemIndex + pageSize,
  );
  const tableRows = visibleVoices.map((voice, index) => ({
    id: `${firstItemIndex + index}-${voice.voiceId}`,
    name: voice.title,
    languages: voice.languages.filter(Boolean).join(', '),
    persona: voice.persona.filter(Boolean).join(', '),
    features: voice.features.filter(Boolean).join(', '),
    voiceId: voice.voiceId,
    preview: voice.previewUrl ? 'Available' : 'Unavailable',
  }));
  const voiceByRowId = new Map(
    tableRows.map((row, index) => [row.id, visibleVoices[index]]),
  );

  const stopPreview = (audio = audioRef.current) => {
    if (!audio || audioRef.current !== audio) return;
    audio.pause();
    audioRef.current = null;
    setPlayingVoiceId(null);
  };

  const togglePreview = (voice: VoiceCatalogItem) => {
    if (!voice.previewUrl) return;
    if (playingVoiceId === voice.voiceId) {
      stopPreview(audioRef.current);
      return;
    }

    audioRef.current?.pause();
    const audio = new Audio(voice.previewUrl);
    audioRef.current = audio;
    setPlayingVoiceId(voice.voiceId);
    audio.onended = () => stopPreview(audio);
    audio.onerror = () => stopPreview(audio);
    audio.play().catch(() => stopPreview(audio));
  };

  const onSearch = (event: '' | ChangeEvent<HTMLInputElement>, value = '') => {
    setQuery(event === '' ? value : event.target.value);
    setPage(1);
    stopPreview();
  };

  return (
    <section
      aria-label="Voice catalog"
      className="flex min-h-0 flex-1 flex-col overflow-hidden"
    >
      <DataTable rows={tableRows} headers={VOICE_TABLE_HEADERS}>
        {({
          rows,
          headers,
          getExpandHeaderProps,
          getExpandedRowProps,
          getHeaderProps,
          getRowProps,
          getTableProps,
        }) => (
          <TableContainer className="flex min-h-0 flex-1 flex-col">
            <TableToolbar>
              <TableToolbarContent>
                <TableToolbarSearch
                  key={queryFromUrl}
                  defaultValue={queryFromUrl}
                  labelText="Search voices"
                  placeholder="Search by voice, language, or use case"
                  onChange={onSearch}
                  onClear={() => onSearch('', '')}
                />
                {actions}
              </TableToolbarContent>
            </TableToolbar>

            {rows.length > 0 ? (
              <div className="min-h-0 flex-1 overflow-auto">
                <Table
                  {...getTableProps()}
                  aria-label="Provider voices"
                  className="w-full min-w-[64rem] table-fixed"
                  data-testid="voice-catalog-table"
                  size="md"
                >
                  <colgroup>
                    {VOICE_TABLE_COLUMN_WIDTHS.map((width, index) => (
                      <col key={index} style={{ width }} />
                    ))}
                  </colgroup>
                  <TableHead>
                    <TableRow>
                      <TableExpandHeader
                        {...getExpandHeaderProps()}
                        aria-label="Expand all voice details"
                      />
                      {headers.map(header => (
                        <TableHeader
                          {...getHeaderProps({ header })}
                          key={header.key}
                        >
                          {header.header}
                        </TableHeader>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {rows.map(row => {
                      const voice = voiceByRowId.get(row.id);
                      if (!voice) return null;

                      return [
                        <TableExpandRow
                          {...getRowProps({ row })}
                          aria-label={`Expand ${voice.title} details`}
                          key={row.id}
                        >
                          {row.cells.map(cell => {
                            if (cell.info.header === 'name') {
                              return (
                                <TableCell key={cell.id}>
                                  <span className="font-semibold text-foreground">
                                    {voice.title}
                                  </span>
                                </TableCell>
                              );
                            }

                            if (cell.info.header === 'languages') {
                              return (
                                <TableCell key={cell.id}>
                                  <MetadataTags values={voice.languages} />
                                </TableCell>
                              );
                            }

                            if (cell.info.header === 'persona') {
                              return (
                                <TableCell key={cell.id}>
                                  <MetadataTags values={voice.persona} />
                                </TableCell>
                              );
                            }

                            if (cell.info.header === 'features') {
                              return (
                                <TableCell key={cell.id}>
                                  <MetadataTags values={voice.features} />
                                </TableCell>
                              );
                            }

                            if (cell.info.header === 'voiceId') {
                              return (
                                <TableCell key={cell.id}>
                                  <div className="flex min-w-0 items-center justify-between gap-2">
                                    <code className="min-w-0 truncate text-xs text-foreground">
                                      {voice.voiceId}
                                    </code>
                                    <CopyButton
                                      align="left"
                                      kind="ghost"
                                      size="sm"
                                      feedback="Copied"
                                      iconDescription={`Copy ${voice.title} voice ID`}
                                      onClick={() =>
                                        navigator.clipboard?.writeText(
                                          voice.voiceId,
                                        )
                                      }
                                    />
                                  </div>
                                </TableCell>
                              );
                            }

                            return (
                              <TableCell key={cell.id}>
                                {voice.previewUrl ? (
                                  <Button
                                    hasIconOnly
                                    kind="ghost"
                                    size="sm"
                                    renderIcon={
                                      playingVoiceId === voice.voiceId
                                        ? Pause
                                        : Play
                                    }
                                    iconDescription={
                                      playingVoiceId === voice.voiceId
                                        ? `Pause ${voice.title}`
                                        : `Preview ${voice.title}`
                                    }
                                    tooltipPosition="left"
                                    onClick={() => togglePreview(voice)}
                                  />
                                ) : (
                                  <span
                                    aria-label="Preview unavailable"
                                    className="text-muted"
                                  >
                                    —
                                  </span>
                                )}
                              </TableCell>
                            );
                          })}
                        </TableExpandRow>,
                        row.isExpanded ? (
                          <TableExpandedRow
                            {...getExpandedRowProps({ row })}
                            colSpan={headers.length + 1}
                            key={`${row.id}-details`}
                          >
                            <div className="px-4 py-5">
                              <p className="text-xs font-semibold text-muted">
                                Description
                              </p>
                              <p className="mt-2 max-w-4xl text-sm leading-5 text-foreground">
                                {voice.description ||
                                  'No description provided.'}
                              </p>
                            </div>
                          </TableExpandedRow>
                        ) : null,
                      ];
                    })}
                  </TableBody>
                </Table>
              </div>
            ) : (
              <EmptyState
                icon={Search}
                title="No voices found"
                subtitle="Try a different voice name, ID, language, or use case."
              />
            )}

            {filteredVoices.length > 0 && (
              <Pagination
                className="mt-auto shrink-0 border-t border-border-subtle"
                id="voice-catalog-pagination"
                totalItems={filteredVoices.length}
                page={page}
                pageSize={pageSize}
                pageSizes={PAGE_SIZES}
                onChange={next => {
                  setPage(next.page);
                  setPageSize(next.pageSize);
                  stopPreview();
                }}
              />
            )}
          </TableContainer>
        )}
      </DataTable>
    </section>
  );
}
