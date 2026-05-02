interface Props {
  status: 'available' | 'downloaded' | null;
  version: string;
  onInstall: () => void;
  onDismiss: () => void;
}

export default function UpdateBanner({ status, version, onInstall, onDismiss }: Props) {
  if (!status) return null;

  const bg = status === 'downloaded' ? '#1a6b2a' : '#1a3d6b';

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, right: 0, zIndex: 9999,
      background: bg, color: '#fff',
      padding: '8px 16px', display: 'flex', alignItems: 'center', gap: 12,
    }}>
      {status === 'available' && (
        <span>Update v{version} available — downloading in background…</span>
      )}
      {status === 'downloaded' && (
        <>
          <span>v{version} ready to install</span>
          <button
            onClick={onInstall}
            style={{ padding: '4px 12px', cursor: 'pointer', borderRadius: 4 }}
          >
            Restart now
          </button>
        </>
      )}
      <button
        onClick={onDismiss}
        style={{
          marginLeft: 'auto', background: 'none', border: 'none',
          color: '#fff', cursor: 'pointer', fontSize: 20, lineHeight: '1',
        }}
        aria-label="Dismiss"
      >
        ×
      </button>
    </div>
  );
}
