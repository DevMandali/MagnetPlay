import videojs from 'video.js';

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
      const dur = isFinite(p.duration() ?? Infinity) ? (p.duration() ?? Infinity) : Infinity;
      const cur = p.currentTime() ?? 0;
      // this.seconds is signed (-10 for back, +10 for forward)
      p.currentTime(Math.max(0, Math.min(dur, cur + this.seconds)));
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
    videojs.Vhs.xhr.beforeRequest = (options: Record<string, unknown>) => {
      options.headers = (options.headers as Record<string, string>) ?? {};
      (options.headers as Record<string, string>)['X-Player'] = 'mp/1.0';
      return options;
    };
  }
}
