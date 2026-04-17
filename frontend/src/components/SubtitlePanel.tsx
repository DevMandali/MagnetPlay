import { useRef, useState } from 'react';
import { SubtitleTrack } from '../types';

interface Props {
  tracks: SubtitleTrack[];
  onAdd: (file: File, label: string, lang: string) => void;
  onToggle: (id: string) => void;
  onRemove: (id: string) => void;
  fontSize: number;
  onFontSizeChange: (v: number) => void;
}

export default function SubtitlePanel({ tracks, onAdd, onToggle, onRemove, fontSize, onFontSizeChange }: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [label, setLabel] = useState('');
  const [lang, setLang] = useState('');

  const handleFilePick = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const ext = file.name.split('.').pop()?.toLowerCase();
    if (ext !== 'srt' && ext !== 'vtt') return;
    onAdd(file, label || file.name.replace(/\.[^.]+$/, ''), lang || 'en');
    setLabel('');
    setLang('');
    e.target.value = '';
  };

  const allOff = !tracks.some(t => t.active);
  const sliderPct = ((fontSize - 50) / 250 * 100);

  return (
    <div className="subtitle-panel">
      <div className="subtitle-panel-header">
        <span className="subtitle-panel-title">⬡ Subtitles &amp; Captions</span>
        <span className="subtitle-panel-hint">Supports .srt and .vtt</span>
      </div>

      <div className="subtitle-upload-row">
        <input
          className="subtitle-field subtitle-field-label"
          placeholder="Track label"
          value={label}
          onChange={e => setLabel(e.target.value)}
        />
        <input
          className="subtitle-field subtitle-field-lang"
          placeholder="en"
          maxLength={10}
          value={lang}
          onChange={e => setLang(e.target.value)}
        />
        <button className="btn-upload" onClick={() => fileInputRef.current?.click()}>
          + Upload
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept=".srt,.vtt"
          style={{ display: 'none' }}
          onChange={handleFilePick}
        />
        <div className="subtitle-formats">
          <span className="fmt-pill">SRT</span>
          <span className="fmt-pill">VTT</span>
        </div>
      </div>

      <div className="subtitle-track-list">
        {tracks.length === 0 && (
          <div className="subtitle-empty">No subtitle tracks loaded</div>
        )}

        <div
          className={`subtitle-off-row${allOff ? ' is-active' : ''}`}
          onClick={() => tracks.filter(t => t.active).forEach(t => onToggle(t.id))}
        >
          <span className="off-label">Off</span>
        </div>

        {tracks.map(t => (
          <div
            key={t.id}
            className={`subtitle-track-row${t.active ? ' is-active' : ''}`}
            onClick={() => onToggle(t.id)}
          >
            <div className="sub-radio">
              <div className="sub-radio-dot" />
            </div>
            <div className="track-info">
              <span className="track-label">{t.label}</span>
              <span className="track-lang-badge">{t.srclang}</span>
              <span className="track-format-badge">{t.format}</span>
            </div>
            <button
              className="track-remove-btn"
              onClick={e => { e.stopPropagation(); onRemove(t.id); }}
            >✕</button>
          </div>
        ))}
      </div>

      <div className="subtitle-size-row">
        <span className="subtitle-size-label">Size</span>
        <span className="subtitle-size-value">{fontSize}%</span>
        <input
          type="range"
          className="subtitle-size-slider"
          min={50}
          max={300}
          step={5}
          value={fontSize}
          style={{ '--pct': `${sliderPct}%` } as React.CSSProperties}
          onChange={e => onFontSizeChange(Number(e.target.value))}
        />
        <button className="subtitle-size-reset" onClick={() => onFontSizeChange(100)}>Reset</button>
      </div>
    </div>
  );
}
