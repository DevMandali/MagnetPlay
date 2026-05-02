// In Electron (file:// protocol) Vite's dev proxy is absent — use absolute URLs.
const IS_FILE = window.location.protocol === 'file:';
export const API_BASE = IS_FILE ? 'http://localhost:8080' : '';
export const HLS_BASE = IS_FILE ? 'http://localhost:8091' : '';

export function srtToVtt(srt: string): string {
  let s = srt.replace(/\r\n?/g, '\n').trim();
  s = s.replace(/(\d{2}:\d{2}:\d{2}),(\d{3})/g, '$1.$2');
  return 'WEBVTT\n\n' + s;
}

export function readFileAsText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as string);
    reader.onerror = () => reject(reader.error);
    reader.readAsText(file, 'utf-8');
  });
}

export async function detectMoovPosition(url: string): Promise<'front' | 'end' | 'unknown'> {
  try {
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), 40000);
    const res = await fetch(url, { headers: { Range: 'bytes=0-11' }, signal: ctrl.signal });
    clearTimeout(timer);
    // Range requests return 206; reject anything else that isn't OK
    if (!res.ok && res.status !== 206) return 'unknown';
    const bytes = new Uint8Array(await res.arrayBuffer());
    if (bytes.length < 8) return 'unknown';
    const boxType = String.fromCharCode(bytes[4], bytes[5], bytes[6], bytes[7]);
    if (boxType === 'ftyp' || boxType === 'moov') return 'front';
    if (['mdat', 'free', 'wide', 'skip', 'pdin'].includes(boxType)) return 'end';
    return 'unknown';
  } catch (err: unknown) {
    if ((err as { name?: string }).name !== 'AbortError') console.warn('[moov-detect]', err);
    return 'unknown';
  }
}
