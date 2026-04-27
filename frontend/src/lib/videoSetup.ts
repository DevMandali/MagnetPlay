import videojs from 'video.js';

// ── Helpers shared by stats popup ────────────────────────────────────────────

function fmtBytes(b: number): string {
  if (b <= 0) return '0 B';
  if (b < 1024) return `${b} B`;
  if (b < 1024 ** 2) return `${(b / 1024).toFixed(1)} KB`;
  if (b < 1024 ** 3) return `${(b / 1024 ** 2).toFixed(1)} MB`;
  return `${(b / 1024 ** 3).toFixed(2)} GB`;
}

function fmtSpeed(bps: number): string {
  return bps <= 0 ? '—' : `${fmtBytes(bps)}/s`;
}

const STATS_SVG = `
  <svg viewBox="0 0 24 24" width="16" height="16" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
    <circle cx="12" cy="12" r="9" stroke="white" stroke-width="1.8" fill="none"/>
    <ellipse cx="12" cy="12" rx="3.5" ry="9" stroke="white" stroke-width="1.5" fill="none"/>
    <line x1="3" y1="12" x2="21" y2="12" stroke="white" stroke-width="1.5"/>
    <line x1="5" y1="7.5" x2="19" y2="7.5" stroke="white" stroke-width="1.3"/>
    <line x1="5" y1="16.5" x2="19" y2="16.5" stroke="white" stroke-width="1.3"/>
  </svg>`;

// ── registerStatsButton ───────────────────────────────────────────────────────

export function registerStatsButton(): void {
  if (videojs.getComponent('StatsButton')) return;

  const Button = videojs.getComponent('Button');

  class StatsButton extends Button {
    private readonly infoHash: string;
    private readonly fileId: string;
    private popup: HTMLElement | null = null;
    private pollInterval: ReturnType<typeof setInterval> | null = null;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    constructor(player: ReturnType<typeof videojs>, options: any) {
      super(player, options);
      // NOTE: field initializers (= null) run after super() with useDefineForClassFields:true.
      // All DOM setup must happen here, after super() and after field initializers fire.
      this.infoHash = options.infoHash ?? '';
      this.fileId   = options.fileId   ?? '';
      this.addClass('vjs-stats-btn');
      this.controlText('Download Status');

      this.popup = document.createElement('div');
      this.popup.className = 'vjs-stats-popup';
      this.popup.innerHTML = '<span class="vsp-loading">Loading…</span>';
      this.popup.style.display = 'none';
      this.el().appendChild(this.popup);

      this.el().addEventListener('mouseenter', () => this.showPopup());
      this.el().addEventListener('mouseleave', () => this.hidePopup());
    }

    createEl() {
      const el = super.createEl('button', {}, { type: 'button' });
      const ph = el.querySelector('.vjs-icon-placeholder');
      if (ph) ph.innerHTML = STATS_SVG;
      return el;
    }

    private showPopup() {
      if (!this.popup) return;
      this.popup.style.display = 'block';
      this.doPoll();
      if (this.pollInterval) clearInterval(this.pollInterval);
      this.pollInterval = setInterval(() => this.doPoll(), 2000);
    }

    private hidePopup() {
      if (!this.popup) return;
      this.popup.style.display = 'none';
      if (this.pollInterval) { clearInterval(this.pollInterval); this.pollInterval = null; }
    }

    private async doPoll() {
      try {
        const res = await fetch(
          `/v1/torrent/stats/${this.infoHash}?fileId=${encodeURIComponent(this.fileId)}`
        );
        if (!res.ok || !this.popup) return;
        const s = await res.json() as {
          totalSize: number; downloadedBytes: number;
          completionPct: number; downloadSpeedBps: number;
          seeders: number; peers: number; trackers: number;
        };
        const pct      = Math.min(Math.max(s.completionPct, 0), 100);
        const complete = pct >= 99.9;
        this.popup.innerHTML = `
          <div class="vsp-title">Download Status</div>
          <div class="vsp-grid">
            <span class="vsp-key">Total</span>
            <span class="vsp-val">${fmtBytes(s.totalSize)}</span>
            <span class="vsp-key">Downloaded</span>
            <span class="vsp-val${complete ? ' ok' : ''}">${fmtBytes(s.downloadedBytes)}</span>
            <span class="vsp-key">Progress</span>
            <span class="vsp-val${complete ? ' ok' : ' accent'}">${pct.toFixed(1)}%</span>
            <span class="vsp-key">Speed</span>
            <span class="vsp-val">${complete ? '— seeding' : fmtSpeed(s.downloadSpeedBps)}</span>
            <span class="vsp-key">Seeders</span>
            <span class="vsp-val">${s.seeders ?? 0}</span>
            <span class="vsp-key">Peers</span>
            <span class="vsp-val">${s.peers ?? 0}</span>
            <span class="vsp-key">Trackers</span>
            <span class="vsp-val">${s.trackers ?? 0}</span>
          </div>
          <div class="vsp-bar">
            <div class="vsp-fill${complete ? ' complete' : ''}" style="width:${pct}%"></div>
          </div>`;
      } catch { /* retain last render on network error */ }
    }

    dispose() {
      this.hidePopup();
      super.dispose();
    }
  }

  videojs.registerComponent('StatsButton', StatsButton);
}

// ── Skip buttons ──────────────────────────────────────────────────────────────

function makeSkipSVG(dir: 'back' | 'forward'): string {
  const isBack = dir === 'back';
  const arc = isBack
    ? 'M 16 6 A 10 10 0 1 0 24.66 11'
    : 'M 16 6 A 10 10 0 1 1 7.34 11';
  const arrow = isBack
    ? 'M 12 3.5 L 16 6 L 13.5 10'
    : 'M 20 3.5 L 16 6 L 18.5 10';
  return `
    <svg viewBox="0 0 32 32" width="22" height="22"
         xmlns="http://www.w3.org/2000/svg"
         aria-hidden="true" focusable="false">
      <path d="${arc}" stroke="white" stroke-width="2.4" fill="none" stroke-linecap="round"/>
      <path d="${arrow}" stroke="white" stroke-width="2.4" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
      <text x="16" y="21" text-anchor="middle" font-size="9.5" font-weight="700"
            font-family="Arial,Helvetica,sans-serif" fill="white">10</text>
    </svg>`;
}

export function registerSkipButtons(): void {
  if (videojs.getComponent('SkipBackButton')) return;

  const Button = videojs.getComponent('Button');

  class SkipButton extends Button {
    private readonly seconds: number;
    private readonly isBack: boolean;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    constructor(player: ReturnType<typeof videojs>, options: any) {
      super(player, options);
      this.seconds = options.seconds ?? 10;
      this.isBack = (options.direction ?? 'forward') === 'back';
      this.addClass('vjs-skip-btn');
      this.controlText(this.isBack ? 'Rewind 10 seconds' : 'Forward 10 seconds');
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    handleClick(e: any) {
      super.handleClick(e);
      const p = this.player();
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const mkvDur: number = (p as any)._mpDuration ?? 0;
      if (mkvDur > 0) {
        // MKV path: raw currentTime starts from 0 after each seek; absolute position = currentTime + _mpSeekOffset.
        // Must set _mpPendingSeek before calling currentTime() so the 'seeking' handler reads the right target.
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const offset: number = (p as any)._mpSeekOffset ?? 0;
        const absoluteNow = (p.currentTime() ?? 0) + offset;
        const target = Math.max(0, Math.min(mkvDur, absoluteNow + this.seconds));
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (p as any)._mpPendingSeek = target;
        p.currentTime(target);
      } else {
        const dur = isFinite(p.duration() ?? Infinity) ? (p.duration() ?? Infinity) : Infinity;
        p.currentTime(Math.max(0, Math.min(dur, (p.currentTime() ?? 0) + this.seconds)));
      }
      p.trigger({ type: 'mp-skip', direction: this.isBack ? 'left' : 'right', seconds: Math.abs(this.seconds) });
    }

    createEl() {
      const el = super.createEl('button', {}, { type: 'button' });
      const ph = el.querySelector('.vjs-icon-placeholder');
      if (ph) ph.innerHTML = '';
      el.insertAdjacentHTML('beforeend', makeSkipSVG(this.options_.seconds < 0 ? 'back' : 'forward'));
      return el;
    }
  }

  videojs.registerComponent('SkipBackButton', class extends SkipButton {
    constructor(player: ReturnType<typeof videojs>, options = {}) {
      super(player, { ...options, direction: 'back', seconds: -10 });
    }
  });

  videojs.registerComponent('SkipForwardButton', class extends SkipButton {
    constructor(player: ReturnType<typeof videojs>, options = {}) {
      super(player, { ...options, direction: 'forward', seconds: +10 });
    }
  });
}

export function configureVhs(): void {
  // @ts-expect-error Vhs is not in @types/video.js
  if (videojs.Vhs) {
    // @ts-expect-error
    videojs.Vhs.xhr.onRequest = (options: Record<string, unknown>) => {
      options.headers = (options.headers as Record<string, string>) ?? {};
      (options.headers as Record<string, string>)['X-Player'] = 'mp/1.0';
      return options;
    };
  }
}

export function registerAudioTrackButton(): void {
  if (videojs.getComponent('AudioTrackMenuButton')) return;

  const MenuButton = videojs.getComponent('MenuButton');
  const MenuItem = videojs.getComponent('MenuItem');

  class AudioTrackMenuItem extends MenuItem {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    constructor(player: ReturnType<typeof videojs>, options: any) {
      super(player, options);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (this as any).selectable = true;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (this as any).isSelected_ = options.selected ?? false;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      if ((this as any).isSelected_) this.addClass('vjs-selected');
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    handleClick(event: any) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (MenuItem.prototype as any).handleClick.call(this, event);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const tracks = this.player().audioTracks() as any;
      for (let i = 0; i < tracks.length; i++) {
        tracks[i].enabled = tracks[i].label === (this as any).options_.label;
      }
    }
  }

  class AudioTrackMenuButton extends MenuButton {
    createEl() {
      const el = super.createEl();
      el.setAttribute('title', 'Audio Track');
      return el;
    }

    createItems() {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const tracks = this.player().audioTracks() as any;
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const items: any[] = [];
      for (let i = 0; i < tracks.length; i++) {
        const track = tracks[i];
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        items.push(new AudioTrackMenuItem(this.player(), {
          label: track.label || track.language || `Track ${i + 1}`,
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          selected: (track as any).enabled,
        }));
      }
      return items;
    }

    buildCSSClass() {
      return `vjs-audio-track-btn ${super.buildCSSClass()}`;
    }
  }

  videojs.registerComponent('AudioTrackMenuItem', AudioTrackMenuItem);
  videojs.registerComponent('AudioTrackMenuButton', AudioTrackMenuButton);
}
