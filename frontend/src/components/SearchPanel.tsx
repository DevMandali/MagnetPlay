import { useState, useCallback, useRef } from 'react';
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

const SearchIcon = () => (
  <svg width="13" height="13" viewBox="0 0 13 13" fill="none" style={{ flexShrink: 0, opacity: 0.35, color: 'var(--muted)' }}>
    <circle cx="5.5" cy="5.5" r="4.5" stroke="currentColor" strokeWidth="1.5"/>
    <line x1="9" y1="9" x2="12" y2="12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
  </svg>
);

const LockIcon = () => (
  <svg width="10" height="10" viewBox="0 0 10 10" fill="none" style={{ flexShrink: 0, marginTop: 2, opacity: 0.4, color: 'var(--muted)' }}>
    <rect x="1.5" y="4.5" width="7" height="5" rx="1" stroke="currentColor" strokeWidth="1.2"/>
    <path d="M3 4.5V3a2 2 0 0 1 4 0v1.5" stroke="currentColor" strokeWidth="1.2"/>
  </svg>
);

function SkeletonRow() {
  return (
    <div style={{
      padding: '11px 14px',
      borderRadius: 6,
      border: '1px solid var(--border)',
      background: 'var(--surface)',
      display: 'flex', flexDirection: 'column', gap: 8,
    }}>
      <div className="skeleton-line" style={{ height: 12, width: '68%' }} />
      <div style={{ display: 'flex', gap: 6 }}>
        <div className="skeleton-line" style={{ height: 10, width: 44, animationDelay: '0.1s' }} />
        <div className="skeleton-line" style={{ height: 10, width: 36, animationDelay: '0.2s' }} />
        <div className="skeleton-line" style={{ height: 10, width: 52, marginLeft: 'auto', animationDelay: '0.15s' }} />
      </div>
    </div>
  );
}

function ResultRow({ r, onSelect }: { r: SearchResult; onSelect: (url: string) => void }) {
  const [hovered, setHovered] = useState(false);
  const canStream = !!r.magnetUrl;

  return (
    <div
      onClick={() => canStream && onSelect(r.magnetUrl)}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      title={canStream ? 'Click to stream' : 'No magnet link available'}
      style={{
        padding: '10px 14px',
        borderRadius: 6,
        border: '1px solid var(--border)',
        borderLeft: `3px solid ${hovered && canStream ? 'var(--accent)' : 'transparent'}`,
        background: hovered && canStream ? 'rgba(230,57,70,.04)' : 'var(--surface)',
        cursor: canStream ? 'pointer' : 'not-allowed',
        opacity: canStream ? 1 : 0.4,
        display: 'flex',
        flexDirection: 'column',
        gap: 7,
        transition: 'border-color 0.15s, background 0.15s',
        boxSizing: 'border-box',
      }}
    >
      {/* title */}
      <div style={{ display: 'flex', alignItems: 'flex-start', gap: 6 }}>
        {!canStream && <LockIcon />}
        <span style={{
          fontFamily: 'var(--sans)', fontSize: 13, fontWeight: 500,
          color: 'var(--text)', lineHeight: 1.4, wordBreak: 'break-word',
          flex: 1,
        }}>
          {r.title}
        </span>
      </div>

      {/* meta */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 5, flexWrap: 'wrap' }}>
        {r.qualityTags.map(tag => (
          <span key={tag} style={{
            fontFamily: 'var(--mono)', fontSize: 8, letterSpacing: '1.5px',
            textTransform: 'uppercase',
            padding: '1px 5px', borderRadius: 3,
            border: '1px solid rgba(230,57,70,.4)',
            color: 'var(--accent)',
            lineHeight: 1.8,
          }}>
            {tag}
          </span>
        ))}
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0 }}>
          <span style={{ fontFamily: 'var(--mono)', fontSize: 10, color: 'var(--muted)' }}>
            {fmtBytes(r.sizeBytes)}
          </span>
          <span style={{ fontFamily: 'var(--mono)', fontSize: 10, color: r.seeders > 0 ? 'var(--ok)' : 'var(--muted)' }}>
            ↑ {r.seeders}
          </span>
          <span style={{ fontFamily: 'var(--mono)', fontSize: 10, color: 'var(--muted)' }}>
            ↔ {r.peers}
          </span>
          {r.indexer && (
            <span style={{
              fontFamily: 'var(--mono)', fontSize: 8, letterSpacing: '1px',
              textTransform: 'uppercase',
              padding: '1px 5px', borderRadius: 3,
              border: '1px solid var(--border)',
              color: 'var(--muted)',
              lineHeight: 1.8,
            }}>
              {r.indexer}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

export default function SearchPanel({ onSelect }: Props) {
  const [query, setQuery]     = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError]     = useState<string | null>(null);
  const [focused, setFocused] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

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
      const items = data.results ?? [];
      setResults(items);
      if (items.length === 0) setError('No results found');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  }, [query]);

  const clearQuery = () => {
    setQuery('');
    setResults([]);
    setError(null);
    inputRef.current?.focus();
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>

      {/* search bar */}
      <div style={{ display: 'flex', gap: 8 }}>
        <div style={{
          flex: 1, display: 'flex', alignItems: 'center', gap: 8,
          background: 'var(--bg)',
          border: `1px solid ${focused ? 'var(--accent)' : 'var(--border)'}`,
          borderRadius: 6,
          padding: '0 12px',
          transition: 'border-color 0.2s',
        }}>
          <SearchIcon />
          <input
            ref={inputRef}
            type="text"
            placeholder="Search torrents — e.g. Dune 2024 1080p"
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSearch()}
            onFocus={() => setFocused(true)}
            onBlur={() => setFocused(false)}
            autoFocus
            style={{
              flex: 1, background: 'transparent', border: 'none', outline: 'none',
              padding: '10px 0',
              fontFamily: 'var(--mono)', fontSize: 12, color: 'var(--text)',
            }}
          />
          {query && (
            <button
              onClick={clearQuery}
              style={{
                background: 'none', border: 'none', cursor: 'pointer',
                color: 'var(--muted)', fontSize: 13, lineHeight: 1,
                padding: '0 2px', flexShrink: 0, transition: 'color 0.15s',
              }}
            >
              ✕
            </button>
          )}
        </div>
        <button
          className="btn btn-primary btn-sm"
          onClick={handleSearch}
          disabled={!query.trim() || loading}
          style={{ flexShrink: 0, minWidth: 80 }}
        >
          {loading
            ? <span className="badge-spin" style={{ display: 'inline-block' }}>↻</span>
            : 'Search'}
        </button>
      </div>

      {/* error */}
      {error && (
        <div className="error-box" style={{ margin: 0, padding: '9px 14px' }}>
          <span>⚠</span>
          <span>{error}</span>
        </div>
      )}

      {/* loading skeletons */}
      {loading && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          <SkeletonRow />
          <SkeletonRow />
          <SkeletonRow />
        </div>
      )}

      {/* results */}
      {!loading && results.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div style={{
            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          }}>
            <span style={{ fontFamily: 'var(--mono)', fontSize: 9, letterSpacing: '2px', textTransform: 'uppercase', color: 'var(--muted)' }}>
              {results.length} result{results.length !== 1 ? 's' : ''}
            </span>
            <span style={{ fontFamily: 'var(--mono)', fontSize: 9, letterSpacing: '1.5px', textTransform: 'uppercase', color: 'var(--muted)', opacity: 0.5 }}>
              ↑ sorted by seeders
            </span>
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 4, maxHeight: 380, overflowY: 'auto' }}>
            {results.map((r, i) => (
              <ResultRow key={i} r={r} onSelect={onSelect} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
