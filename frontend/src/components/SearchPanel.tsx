import { useState, useCallback } from 'react';
import { SearchResult, SearchResultsResponse } from '../types';

interface Props {
  onSelect: (magnetUrl: string) => void;
}

function fmtBytes(bytes: number): string {
  if (bytes === 0) return '—';
  if (bytes >= 1e9) return (bytes / 1e9).toFixed(2) + ' GB';
  if (bytes >= 1e6) return (bytes / 1e6).toFixed(1) + ' MB';
  return Math.round(bytes / 1e3) + ' KB';
}

export default function SearchPanel({ onSelect }: Props) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = useCallback(async () => {
    const q = query.trim();
    if (!q) return;
    setLoading(true);
    setError(null);
    setResults([]);
    try {
      const res = await fetch(`/v1/search?q=${encodeURIComponent(q)}`, {
        signal: AbortSignal.timeout(30000),
      });
      if (res.status === 503) throw new Error('Search unavailable — Prowlarr not running');
      if (!res.ok) throw new Error(`Search error: ${res.status}`);
      const data: SearchResultsResponse = await res.json();
      setResults(data.results);
      if (data.results.length === 0) setError('No results found');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  }, [query]);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      <div style={{ display: 'flex', gap: 8 }}>
        <input
          type="text"
          placeholder="Search torrents… (e.g. Dune 2024 1080p)"
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && handleSearch()}
          style={{ flex: 1, fontFamily: 'var(--mono)', fontSize: 12 }}
          autoFocus
        />
        <button
          className="btn btn-primary"
          onClick={handleSearch}
          disabled={!query.trim() || loading}
        >
          {loading ? '…' : '⌕ Search'}
        </button>
      </div>

      {error && (
        <div style={{
          padding: '8px 12px',
          background: 'rgba(230,57,70,.1)',
          border: '1px solid rgba(230,57,70,.3)',
          borderRadius: 6,
          fontSize: 11,
          color: '#ff8080',
          fontFamily: 'var(--mono)',
        }}>
          ⚠ {error}
        </div>
      )}

      {results.length > 0 && (
        <div style={{
          display: 'flex', flexDirection: 'column', gap: 4,
          maxHeight: 300, overflowY: 'auto',
        }}>
          {results.map((r, i) => (
            <div
              key={i}
              onClick={() => r.magnetUrl && onSelect(r.magnetUrl)}
              title={r.magnetUrl ? 'Click to use this torrent' : 'No magnet link available'}
              style={{
                padding: '8px 12px',
                background: 'var(--surface)',
                border: '1px solid var(--border)',
                borderRadius: 6,
                cursor: r.magnetUrl ? 'pointer' : 'default',
                opacity: r.magnetUrl ? 1 : 0.45,
                display: 'flex',
                flexDirection: 'column',
                gap: 4,
              }}
            >
              <div style={{ fontSize: 12, fontWeight: 500, wordBreak: 'break-word', lineHeight: 1.3 }}>
                {r.title}
              </div>
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', alignItems: 'center' }}>
                {r.qualityTags.map(tag => (
                  <span
                    key={tag}
                    style={{
                      padding: '1px 5px', borderRadius: 3,
                      background: 'var(--accent-dim, rgba(99,102,241,.15))',
                      color: 'var(--accent, #818cf8)',
                      fontSize: 9, fontFamily: 'var(--mono)',
                    }}
                  >
                    {tag}
                  </span>
                ))}
                <span style={{ marginLeft: 'auto', fontSize: 10, color: 'var(--muted)' }}>
                  {fmtBytes(r.sizeBytes)}
                </span>
                <span style={{ fontSize: 10, color: r.seeders > 0 ? '#4ade80' : 'var(--muted)' }}>
                  ↑{r.seeders}
                </span>
                <span style={{ fontSize: 10, color: 'var(--muted)' }}>↔{r.peers}</span>
                {r.indexer && (
                  <span style={{ fontSize: 10, color: 'var(--muted)' }}>[{r.indexer}]</span>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
