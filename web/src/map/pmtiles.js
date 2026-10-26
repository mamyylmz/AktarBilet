// Self-hosted PMTiles v3 client with HTTP 206 Range request verification
export class PMTilesSource {
  constructor(url = '/tiles/izmir.pmtiles') {
    this.url = url;
    this.header = null;
    this.metadata = null;
    this.isRangeSupported = false;
  }

  async init() {
    try {
      // Test HTTP 206 Partial Content Range Request
      const res = await fetch(this.url, {
        headers: { Range: 'bytes=0-126' },
      });
      if (res.status === 206 || res.status === 200) {
        this.isRangeSupported = res.status === 206;
        const buf = await res.arrayBuffer();
        const view = new DataView(buf);
        const magic = String.fromCharCode(
          view.getUint8(0), view.getUint8(1), view.getUint8(2),
          view.getUint8(3), view.getUint8(4), view.getUint8(5), view.getUint8(6)
        );
        if (magic === 'PMTiles') {
          this.header = {
            version: view.getUint8(7),
            rootDirOffset: Number(view.getBigUint64(8, true)),
            rootDirLen: Number(view.getBigUint64(16, true)),
            metaOffset: Number(view.getBigUint64(24, true)),
            metaLen: Number(view.getBigUint64(32, true)),
            minZoom: view.getUint8(100),
            maxZoom: view.getUint8(101),
          };
          if (this.header.metaLen > 0) {
            const metaRes = await fetch(this.url, {
              headers: { Range: `bytes=${this.header.metaOffset}-${this.header.metaOffset + this.header.metaLen - 1}` },
            });
            if (metaRes.ok || metaRes.status === 206) {
              const metaText = await metaRes.text();
              try {
                this.metadata = JSON.parse(metaText);
              } catch (_) {}
            }
          }
        }
      }
    } catch (e) {
      console.warn('PMTiles Range fetch warning:', e);
    }
    return this;
  }
}

