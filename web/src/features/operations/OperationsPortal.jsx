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
  liveTransitError = false,
  onRefreshTransit,
  liveTransitMeta,
  anomalies = [],
}) {
  const [activeTab, setActiveTab] = useState('map'); // 'map' | 'anomalies' | 'telemetry'

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
              <div className="flex flex-wrap gap-2">
                <Button
                  variant={selectedLine === '800' ? 'primary' : 'outline'}
                  size="sm"
                  className="min-h-[44px]"
                  onClick={() => onSelectLine('800')}
                >
                  800 Menemen-Bornova
                </Button>
                <Button
                  variant={selectedLine === '470' ? 'primary' : 'outline'}
                  size="sm"
                  className="min-h-[44px]"
                  onClick={() => onSelectLine('470')}
                >
                  470 Tınaztepe-Lozan
                </Button>
                <Button
                  variant={selectedLine === '35' ? 'primary' : 'outline'}
                  size="sm"
                  className="min-h-[44px]"
                  onClick={() => onSelectLine('35')}
                >
                  35 Ballıkuyu-Konak
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  className="min-h-[44px]"
                  onClick={onRefreshTransit}
                  loading={liveTransitLoading}
                  disabled={liveTransitLoading}
                  aria-label={`${selectedLine} hattı araç konumlarını yenile`}
                >
                  Yenile
                </Button>
              </div>
            }
          >
            {liveTransitLoading && (
              <p className="mb-3 text-sm text-muted" role="status" aria-live="polite">
                {selectedLine} hattının araç konumları yükleniyor…
              </p>
            )}
            {liveTransitError && (
              <div role="alert" className="mb-3 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-danger/30 bg-surface-muted p-4">
                <p className="text-sm text-danger">Canlı araç verisi alınamadı. Tekrar deneyebilirsiniz.</p>
                <Button variant="outline" size="sm" className="min-h-[44px]" onClick={onRefreshTransit}>
                  Tekrar dene
                </Button>
              </div>
            )}
            {liveTransitMeta?.is_stale && !liveTransitError && (
              <p className="mb-3 text-sm text-warning" role="status">
                ESHOT yanıt vermedi; son alınan araç verisi gösteriliyor.
              </p>
            )}
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
        <div className="grid min-w-0 grid-cols-1 gap-4 xl:grid-cols-2">
          <Card
            title="Validatör Cihaz Durumları"
            subtitle="Bu görünüm yalnızca backend'den doğrulanmış cihaz telemetrisi gösterir"
          >
            <div className="rounded-lg border border-border bg-surface-muted p-4" role="status">
              <span className="badge badge-warning">Telemetri verisi yok</span>
              <p className="mt-2 text-sm text-muted">Bu API'de cihaz heartbeat'i, son sıra numarası veya son veri zamanı sunulmuyor. Çevrimiçi durumu ve hat eşlemesi gösterilmiyor.</p>
            </div>
          </Card>

          <Card
            title="WAL & Kuyruk Durumu"
            subtitle="Çevrimdışı paket senkronizasyonu ve disk günlüğü telemetrisi"
          >
            <div className="space-y-4">
              <div className="rounded-lg border border-border bg-surface-muted p-4" role="status">
                <span className="badge badge-warning">Durum doğrulanmadı</span>
                <p className="mt-2 text-sm text-muted">Validator WAL ve kuyruk ölçümleri bu API'den alınamıyor; sayı veya kalıcılık durumu tahmin edilmiyor.</p>
              </div>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
