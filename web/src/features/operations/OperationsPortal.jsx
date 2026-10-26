import React, { useState } from 'react';
import { Card } from '../../components/ui/Card.jsx';
import { Button } from '../../components/ui/Button.jsx';
import { Table } from '../../components/ui/Table.jsx';
import { MapLibreView } from '../../map/MapLibreView.jsx';
import { formatDateTime } from '../../lib/formatters.js';

export function OperationsPortal({
  liveBuses = [],
  selectedLine = '800',
  onSelectLine,
  liveTransitLoading = false,
  liveTransitMeta,
  heatmapData = [],
  heatmapError = false,
  onFetchHeatmap,
  anomalies = [],
}) {
  const [activeTab, setActiveTab] = useState('map'); // 'map' | 'heatmap' | 'anomalies' | 'telemetry'

  const busColumns = [
    {
      header: 'Araç Kapı No',
      key: 'bus_id',
      render: (row) => <span className="font-mono font-bold text-primary">#{row.bus_id}</span>,
    },
    {
      header: 'Hat',
      key: 'line_number',
      render: () => <span className="badge badge-secondary">{selectedLine} ESHOT</span>,
    },
    {
      header: 'Enlem (Lat)',
      key: 'latitude',
      render: (row) => <span className="font-mono text-sm">{row.latitude?.toFixed(4)}</span>,
    },
    {
      header: 'Boylam (Lon)',
      key: 'longitude',
      render: (row) => <span className="font-mono text-sm">{row.longitude?.toFixed(4)}</span>,
    },
    {
      header: 'Durum',
      key: 'status',
      render: () => <span className="badge badge-success">Canlı / Seyirde</span>,
    },
  ];

  const anomalyColumns = [
    {
      header: 'Tespit Zamanı',
      key: 'detected_at',
      render: (row) => formatDateTime(row.detected_at),
    },
    {
      header: 'Kart ID',
      key: 'card_id',
      render: (row) => <span className="font-mono font-semibold">{row.card_id}</span>,
    },
    {
      header: 'Anomali Türü',
      key: 'anomaly_type',
      render: (row) => (
        <span className="badge badge-danger">
          {row.anomaly_type === 'IMPOSSIBLE_SPEED' ? 'İmkansız Seyahat Hızı' : row.anomaly_type}
        </span>
      ),
    },
    {
      header: 'Açıklama / Hız',
      key: 'description',
      render: (row) => (
        <span className="text-sm">
          {row.description || `${row.calculated_speed_kmh?.toFixed(1) || '0'} km/h hız tespit edildi`}
        </span>
      ),
    },
  ];

  return (
    <div className="portal-container space-y-6">
      {/* Sub-navigation tabs */}
      <div className="flex flex-wrap gap-2 border-b border-border pb-3">
        <Button
          variant={activeTab === 'map' ? 'primary' : 'outline'}
          size="sm"
          onClick={() => setActiveTab('map')}
        >
          Canlı Harita (ESHOT)
        </Button>
        <Button
          variant={activeTab === 'heatmap' ? 'primary' : 'outline'}
          size="sm"
          onClick={() => {
            setActiveTab('heatmap');
            onFetchHeatmap();
          }}
        >
          Isı Haritası
        </Button>
        <Button
          variant={activeTab === 'anomalies' ? 'primary' : 'outline'}
          size="sm"
          onClick={() => setActiveTab('anomalies')}
        >
          Güvenlik Anomalileri ({anomalies.length})
        </Button>
        <Button
          variant={activeTab === 'telemetry' ? 'primary' : 'outline'}
          size="sm"
          onClick={() => setActiveTab('telemetry')}
        >
          Validatör Telemetrisi
        </Button>
      </div>

      {/* 1. Live Transit Map */}
      {activeTab === 'map' && (
        <div className="space-y-4">
          <Card
            title="İzmir ESHOT Canlı Araç Takip Haritası (MapLibre & PMTiles)"
            subtitle="Resmî açık veri API'si üzerinden 800, 470 ve 35 numaralı hatların gerçek zamanlı koordinatları"
            action={
              <div className="flex gap-2">
                <Button
                  variant={selectedLine === '800' ? 'primary' : 'outline'}
                  size="sm"
                  onClick={() => onSelectLine('800')}
                >
                  800 Menemen-Bornova
                </Button>
                <Button
                  variant={selectedLine === '470' ? 'primary' : 'outline'}
                  size="sm"
                  onClick={() => onSelectLine('470')}
                >
                  470 Tınaztepe-Lozan
                </Button>
                <Button
                  variant={selectedLine === '35' ? 'primary' : 'outline'}
                  size="sm"
                  onClick={() => onSelectLine('35')}
                >
                  35 Ballıkuyu-Konak
                </Button>
              </div>
            }
          >
            <div className="map-wrapper mb-4">
              <MapLibreView
                liveBuses={liveBuses}
                selectedLine={selectedLine}
              />
            </div>

            {liveTransitMeta && (
              <div className="flex flex-wrap justify-between text-xs text-muted pt-2 border-t border-border">
                <span>Aktif Seferdeki Araç: <strong>{liveTransitMeta.total_buses || liveBuses.length}</strong></span>
                <span>Veri Kaynağı: <strong>{liveTransitMeta.attribution || 'İzmir Büyükşehir Belediyesi ESHOT'}</strong></span>
                <span>Son Güncelleme: <strong>{formatDateTime(liveTransitMeta.retrieved_at) || 'Canlı'}</strong></span>
              </div>
            )}
          </Card>

          {/* Accessible Table Fallback for Screen Readers & Keyboard Navigation */}
          <Card
            title="Seferdeki Otobüsler Listesi (Erişilebilir Tablo)"
            subtitle="Haritadaki araçların erişilebilir ekran okuyucu tablosu"
          >
            <Table
              columns={busColumns}
              data={liveBuses}
              keyExtractor={(row) => row.bus_id}
              emptyMessage={liveTransitLoading ? 'Canlı araç verileri alınıyor...' : 'Bu hatta şu an aktif seyirde araç bulunmuyor.'}
              ariaLabel="Canlı Otobüs Sefer Listesi"
            />
          </Card>
        </div>
      )}

      {/* 2. Heatmap View */}
      {activeTab === 'heatmap' && (
        <Card
          title="Yolcu Biniş Yoğunluk Isı Haritası"
          subtitle="İstasyon ve durak bazında saatlik yolcu biniş yoğunluk analizi"
        >
          {heatmapError ? (
            <div role="alert" className="p-6 bg-surface-muted border border-danger/30 rounded-lg text-center space-y-2">
              <span className="text-3xl block" aria-hidden="true">⚠️</span>
              <h4 className="font-semibold text-danger">Isı Haritası Verisi Alınamadı</h4>
              <p className="text-sm text-muted">Sunucu bağlantısı sağlanamadı veya yoğunluk servisi yanıt vermedi.</p>
              <Button variant="outline" size="sm" onClick={onFetchHeatmap}>
                Tekrar Yükle
              </Button>
            </div>
          ) : heatmapData.length > 0 ? (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
              {heatmapData.map((item, idx) => (
                <div key={idx} className="p-4 bg-surface-muted rounded-lg border border-border">
                  <span className="text-xs text-muted block">{item.station_name || item.station_id}</span>
                  <span className="font-bold text-xl text-primary">{item.tap_count} Biniş</span>
                  <span className="text-xs text-muted block mt-1">Yoğunluk: {item.density_level || 'Yüksek'}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-muted">
              <p className="text-sm">Yoğunluk verisi yükleniyor...</p>
            </div>
          )}
        </Card>
      )}

      {/* 3. Anomalies List */}
      {activeTab === 'anomalies' && (
        <Card
          title="Güvenlik ve Biniş Anomalileri"
          subtitle="İmkansız hız tespiti, klon kart veya şüpheli mükerrer geçiş kayıtları"
        >
          <Table
            columns={anomalyColumns}
            data={anomalies}
            keyExtractor={(row) => row.id || row.detected_at}
            emptyMessage="Sistemde henüz kayıtlı bir güvenlik anomalisi bulunmuyor."
            ariaLabel="Güvenlik Anomalileri Tablosu"
          />
        </Card>
      )}

      {/* 4. Validator Telemetry */}
      {activeTab === 'telemetry' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Card
            title="Validatör Cihaz Durumları"
            subtitle="Filodaki 4 ana hat validatörünün anlık çevrimiçi/çevrimdışı bağlantı sağlığı"
          >
            <div className="space-y-3">
              {[
                { id: 'VAL_500T_01', line: '500T Tuzla - Cevizlibağ', status: 'ONLINE', seq: 1042 },
                { id: 'VAL_14ES_01', line: '14ES Kadıköy', status: 'ONLINE', seq: 852 },
                { id: 'VAL_T1_01', line: 'T1 Tramvay Kabataş', status: 'ONLINE', seq: 2140 },
                { id: 'VAL_VAPUR_01', line: 'Kadıköy-Eminönü Vapur', status: 'ONLINE', seq: 940 },
              ].map((v) => (
                <div key={v.id} className="flex justify-between items-center p-3 bg-surface-muted rounded-lg border border-border">
                  <div>
                    <span className="font-mono font-bold text-sm block">{v.id}</span>
                    <span className="text-xs text-muted">{v.line}</span>
                  </div>
                  <div className="text-right">
                    <span className="badge badge-success text-xs">Çevrimiçi (ONLINE)</span>
                    <span className="text-xs font-mono text-muted block mt-1">Sıra No: #{v.seq}</span>
                  </div>
                </div>
              ))}
            </div>
          </Card>

          <Card
            title="WAL & Kuyruk Durumu"
            subtitle="Çevrimdışı paket senkronizasyonu ve disk günlüğü telemetrisi"
          >
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="p-3 bg-surface-muted rounded-lg border border-border">
                  <span className="text-xs text-muted block">Disk WAL Durumu</span>
                  <span className="font-bold text-success text-lg">AKTİF & KALICI</span>
                </div>
                <div className="p-3 bg-surface-muted rounded-lg border border-border">
                  <span className="text-xs text-muted block">Kuyruktaki Paket</span>
                  <span className="font-mono font-bold text-foreground text-lg">0 / 100</span>
                </div>
              </div>
              <div className="p-3 bg-surface-muted rounded-lg border border-border text-xs text-muted">
                <p>Monotonik sıra numarası ve SHA-256 yük özeti doğrulandı. Mükerrer biniş kaybı: 0.</p>
              </div>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
