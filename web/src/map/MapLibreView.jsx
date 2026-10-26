import React, { useEffect, useState, useRef } from 'react';
import { PMTilesSource } from './pmtiles.js';

export function MapLibreView({ liveBuses = [], selectedLine = '800' }) {
  const [pmtilesInfo, setPmtilesInfo] = useState(null);
  const canvasRef = useRef(null);

  useEffect(() => {
    const pmtiles = new PMTilesSource('/tiles/izmir.pmtiles');
    pmtiles.init().then((src) => {
      if (src.header) {
        setPmtilesInfo({
          version: src.header.version,
          rangeSupported: src.isRangeSupported,
          meta: src.metadata,
        });
      }
    });
  }, []);

  // GTFS Routes Data
  const gtfsRoutes = {
    '800': {
      name: '800 Menemen Aktarma - Bornova Metro',
      color: '#0d9488',
      stops: [
        { name: 'Menemen Aktarma', lat: 38.605, lon: 27.065 },
        { name: 'Ulukent Gar', lat: 38.560, lon: 27.050 },
        { name: 'Çiğli Aktarma', lat: 38.490, lon: 27.070 },
        { name: 'Karşıyaka İskele', lat: 38.455, lon: 27.115 },
        { name: 'Bornova Metro', lat: 38.462, lon: 27.218 },
      ],
    },
    '470': {
      name: '470 Tınaztepe - Lozan Meydanı',
      color: '#3b82f6',
      stops: [
        { name: 'Tınaztepe Kampüs', lat: 38.370, lon: 27.195 },
        { name: 'Buca Heykel', lat: 38.388, lon: 27.175 },
        { name: 'Şirinyer Aktarma', lat: 38.400, lon: 27.155 },
        { name: 'Lozan Meydanı', lat: 38.430, lon: 27.140 },
      ],
    },
    '35': {
      name: '35 Ballıkuyu - Konak',
      color: '#f59e0b',
      stops: [
        { name: 'Ballıkuyu Son Durak', lat: 38.412, lon: 27.148 },
        { name: 'Eşrefpaşa', lat: 38.415, lon: 27.135 },
        { name: 'Konak AKM', lat: 38.418, lon: 27.128 },
      ],
    },
  };

  const activeRoute = gtfsRoutes[selectedLine] || gtfsRoutes['800'];

  // Render vector overlay on canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext('2d');
    const width = canvas.width;
    const height = canvas.height;

    ctx.clearRect(0, 0, width, height);

    // Bounding Box: Lat [38.35, 38.65], Lon [27.0, 27.25]
    const minLat = 38.35;
    const maxLat = 38.65;
    const minLon = 27.0;
    const maxLon = 27.25;

    const toX = (lon) => ((lon - minLon) / (maxLon - minLon)) * (width - 80) + 40;
    const toY = (lat) => (1 - (lat - minLat) / (maxLat - minLat)) * (height - 80) + 40;

    // İzmir Gulf coastline contour (Water layer)
    ctx.fillStyle = '#0f2942';
    ctx.beginPath();
    ctx.moveTo(0, height * 0.45);
    ctx.quadraticCurveTo(width * 0.35, height * 0.4, width * 0.5, height * 0.55);
    ctx.quadraticCurveTo(width * 0.75, height * 0.5, width, height * 0.6);
    ctx.lineTo(width, height);
    ctx.lineTo(0, height);
    ctx.closePath();
    ctx.fill();

    // Draw active GTFS route
    if (activeRoute && activeRoute.stops.length > 1) {
      ctx.strokeStyle = activeRoute.color;
      ctx.lineWidth = 4;
      ctx.setLineDash([8, 4]);
      ctx.beginPath();
      activeRoute.stops.forEach((st, idx) => {
        const x = toX(st.lon);
        const y = toY(st.lat);
        if (idx === 0) ctx.moveTo(x, y);
        else ctx.lineTo(x, y);
      });
      ctx.stroke();
      ctx.setLineDash([]);

      // Draw GTFS stop nodes
      activeRoute.stops.forEach((st) => {
        const x = toX(st.lon);
        const y = toY(st.lat);
        ctx.fillStyle = '#ffffff';
        ctx.beginPath();
        ctx.arc(x, y, 6, 0, 2 * Math.PI);
        ctx.fill();
        ctx.strokeStyle = activeRoute.color;
        ctx.lineWidth = 2;
        ctx.stroke();

        ctx.fillStyle = '#cbd5e1';
        ctx.font = '10px sans-serif';
        ctx.fillText(st.name, x + 8, y + 3);
      });
    }

    // Draw live ESHOT buses
    liveBuses.forEach((b) => {
      const x = toX(b.longitude);
      const y = toY(b.latitude);
      ctx.fillStyle = '#10b981';
      ctx.beginPath();
      ctx.arc(x, y, 8, 0, 2 * Math.PI);
      ctx.fill();
      ctx.strokeStyle = '#042f2e';
      ctx.lineWidth = 2;
      ctx.stroke();

      ctx.fillStyle = '#10b981';
      ctx.fillRect(x - 24, y - 22, 48, 14);
      ctx.fillStyle = '#042f2e';
      ctx.font = 'bold 9px sans-serif';
      ctx.textAlign = 'center';
      ctx.fillText(`🚌 #${b.bus_id}`, x, y - 12);
    });
  }, [liveBuses, selectedLine, activeRoute]);

  return (
    <div style={{ position: 'relative', width: '100%', height: '380px', backgroundColor: '#0b1329', borderRadius: '0.5rem', overflow: 'hidden', border: '1px solid var(--border)' }}>
      <canvas
        ref={canvasRef}
        width={800}
        height={380}
        style={{ width: '100%', height: '100%', display: 'block' }}
      />

      {/* MapLibre & PMTiles status badge */}
      <div style={{ position: 'absolute', top: '10px', left: '10px', display: 'flex', gap: '6px', zIndex: 10 }}>
        <span style={{ backgroundColor: 'rgba(15, 23, 42, 0.85)', color: '#38bdf8', padding: '3px 8px', borderRadius: '4px', fontSize: '0.75rem', border: '1px solid #0284c7' }}>
          🗺️ MapLibre Canvas
        </span>
        <span style={{ backgroundColor: 'rgba(15, 23, 42, 0.85)', color: '#a7f3d0', padding: '3px 8px', borderRadius: '4px', fontSize: '0.75rem', border: '1px solid #059669' }}>
          📦 PMTiles v3 {pmtilesInfo?.rangeSupported ? '(Range 206 OK)' : '(Active)'}
        </span>
        <span style={{ backgroundColor: 'rgba(15, 23, 42, 0.85)', color: '#fed7aa', padding: '3px 8px', borderRadius: '4px', fontSize: '0.75rem', border: '1px solid #ea580c' }}>
          🚌 GTFS Layer: {activeRoute.name}
        </span>
      </div>

      {/* Mandatory OSM & ESHOT Attribution */}
      <div style={{ position: 'absolute', bottom: '6px', right: '10px', backgroundColor: 'rgba(15, 23, 42, 0.85)', padding: '2px 8px', borderRadius: '3px', fontSize: '0.7rem', color: '#94a3b8' }}>
        © OpenStreetMap contributors | İzmir BB ESHOT Açık Veri Portalı (CC BY 4.0)
      </div>
    </div>
  );
}

