import { useRef, useEffect } from 'react';

interface Props {
  side: 'left' | 'right';
  seconds: number;
  animKey: number;
  onHidden: () => void;
}

const ChevronSVG = ({ delay, side }: { delay: number; side: 'left' | 'right' }) => {
  const d = side === 'right'
    ? 'M 5 4 L 11 10 L 5 16'
    : 'M 11 4 L 5 10 L 11 16';
  return (
    <svg className="nf-chevron"
         viewBox="0 0 16 20" width="16" height="20" fill="none"
         stroke="white" strokeWidth="2.8" strokeLinecap="round" strokeLinejoin="round"
         style={{ animationDelay: `${delay}s`, filter: 'drop-shadow(0 0 3px rgba(255,255,255,.4))' }}>
      <path d={d} />
    </svg>
  );
};

export default function NetflixSkipOverlay({ side, seconds, animKey, onHidden }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    // Remove both classes, force reflow to restart CSS animations, then show
    el.classList.remove('hiding', 'show');
    void el.offsetWidth;
    el.classList.add('show');

    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      el.classList.remove('show');
      el.classList.add('hiding');
    }, 900);

    return () => { if (timerRef.current) clearTimeout(timerRef.current); };
  }, [animKey]);

  const handleTransitionEnd = (e: React.TransitionEvent) => {
    if (e.propertyName === 'opacity' && ref.current?.classList.contains('hiding')) {
      ref.current.classList.remove('hiding');
      onHidden();
    }
  };

  return (
    <div
      ref={ref}
      className={`nf-skip nf-skip-${side}`}
      onTransitionEnd={handleTransitionEnd}
    >
      <div className="nf-oval" />
      {/* key=animKey forces chevron re-mount so animations replay on consecutive presses */}
      <div className="nf-body" key={animKey}>
        <div className="nf-chevrons">
          <ChevronSVG delay={0.00} side={side} />
          <ChevronSVG delay={0.07} side={side} />
          <ChevronSVG delay={0.14} side={side} />
        </div>
        <span className="nf-label">{seconds} second{seconds !== 1 ? 's' : ''}</span>
      </div>
    </div>
  );
}
