import React, { useEffect, useState } from 'react';
import { Card } from '../../components/ui/Card.jsx';
import { Button } from '../../components/ui/Button.jsx';
import { Table } from '../../components/ui/Table.jsx';
import { formatKurus, formatDateTime } from '../../lib/formatters.js';

// Hat ID'den kullanıcı dostu isim üretir; uydurma isim kullanmaz
function friendlyLineName(lineId) {
  if (!lineId) return null;
  const upper = lineId.toUpperCase();
  // Bilinen hat eşlemeleri (seed verisiyle tutarlı)
  const known = {
    LINE_5: '5 Narlıdere–Üçkuyular İskele',
    LINE_10: '10 ESHOT',
    LINE_15: '15 ESHOT',
    LINE_16: '16 ESHOT',
    LINE_TRAM_KONAK: 'Konak Tramvayı',
    LINE_IZDENIZ_BOSTANLI_KONAK: 'İZDENİZ Bostanlı–Konak',
  };
  // Tam eşleme
  if (known[upper]) return known[upper];
  // Prefix eşleme
  for (const [key, val] of Object.entries(known)) {
    if (upper.startsWith(key)) return val;
  }
  return null; // Bilinmeyen hat — raw id gösterilir
}

function friendlyDescription(row) {
  // Ham açıklamayı kullanıcıya uygun forma dönüştür
  const desc = row.description || '';
  const type = row.entry_type;

  if (type === 'dolum' || row.amount_kurus > 0) {
    return 'Bakiye Yükleme';
  }
  if (type === 'ceza') {
    return 'Ceza Kesintisi';
  }
  if (type === 'storno') {
    return 'Storno Düzeltme';
  }

  // Biniş açıklamasından hat adı çıkar: "LINE_500T binişi (Seq: ...)" -> "500T Narlıdere–Konak • Biniş ücreti"
  const lineMatch = desc.match(/^(LINE_\S+)\s+binişi/i);
  if (lineMatch) {
    const friendly = friendlyLineName(lineMatch[1]);
    if (friendly) return `${friendly} • Biniş ücreti`;
    // Hat bilinmiyorsa ham line_id'yi göster
    return `${lineMatch[1]} • Biniş ücreti`;
  }

  // row.line_id varsa (API'den gelmişse)
  if (row.line_id) {
    const friendly = friendlyLineName(row.line_id);
    if (friendly) return `${friendly} • Biniş ücreti`;
  }

  return desc || 'Toplu Taşıma Geçişi';
}

const PAGE_SIZE = 10;

export function PassengerPortal({
  card,
  ledger = [],
  onRecharge,
  onTapTurnstile,
  rechargeLoading = false,
  tapLoading = false,
  cardDataLoading = false,
  cardDataError = '',
  transferTimer = 0,
  transferLevel = 0,
}) {
  const [selectedAmount, setSelectedAmount] = useState(50);
  const [customAmount, setCustomAmount] = useState('');
  const [ledgerPage, setLedgerPage] = useState(0);

  const formatTimer = (secs) => {
    const mins = Math.floor(secs / 60);
    const s = secs % 60;
    return `${mins.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  };

  // Sayfalama
  const totalPages = Math.ceil(ledger.length / PAGE_SIZE);
  const pagedLedger = ledger.slice(ledgerPage * PAGE_SIZE, (ledgerPage + 1) * PAGE_SIZE);
  useEffect(() => setLedgerPage(0), [ledger]);

  const ledgerColumns = [
    {
      header: 'Tarih / Saat',
      key: 'created_at',
      render: (row) => <span className="text-sm font-medium">{formatDateTime(row.created_at)}</span>,
    },
    {
      header: 'İşlem Türü',
      key: 'entry_type',
      render: (row) => {
        if (row.entry_type === 'dolum' || row.amount_kurus > 0) {
          return <span className="badge badge-success">Bakiye Yükleme</span>;
        }
        if (row.entry_type === 'ceza') {
          return <span className="badge badge-danger">Ceza Kesintisi</span>;
        }
        if (row.entry_type === 'storno') {
          return <span className="badge badge-warning">Storno</span>;
        }
        return <span className="badge badge-primary">Turnike Geçişi</span>;
      },
    },
    {
      header: 'Tutar',
      key: 'amount_kurus',
      align: 'right',
      render: (row) => {
        const isCredit = row.amount_kurus > 0;
        return (
          <span className={`font-mono font-semibold ${isCredit ? 'text-success' : 'text-foreground'}`}>
            {isCredit ? '+' : ''}{formatKurus(row.amount_kurus)}
          </span>
        );
      },
    },
    {
      header: 'İşlem Sonrası Bakiye',
      key: 'balance_after_kurus',
      align: 'right',
      render: (row) => (
        <span className="font-mono text-muted">
          {formatKurus(row.balance_after_kurus)}
        </span>
      ),
    },
    {
      header: 'Açıklama',
      key: 'description',
      render: (row) => (
        <span className="text-sm text-foreground">{friendlyDescription(row)}</span>
      ),
    },
  ];

  return (
    <div className="portal-container space-y-6">
      {/* Top Banner / Card Overview */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Visual Transit Card */}
        <div className="md:col-span-1">
          <div className="transit-virtual-card">
            <div className="card-top-row">
              <span className="card-chip-icon" aria-hidden="true" />
              <span className="card-type-badge font-mono uppercase">{card?.card_type ? `${card.card_type} KART` : 'Kart bilgisi bekleniyor'}</span>
            </div>
            <div className="card-middle-row my-4">
              <span className="card-balance-label text-xs uppercase tracking-wider text-white/80">Güncel Bakiye</span>
              <div className="card-balance-value font-mono text-3xl font-bold text-white mt-1">
                {card?.balance_kurus !== null && card?.balance_kurus !== undefined
                  ? formatKurus(card.balance_kurus)
                  : cardDataLoading ? 'Yükleniyor…' : 'Bakiye alınamadı'}
              </div>
            </div>
            <div className="card-bottom-row flex justify-between items-end text-xs text-white/90">
              <div>
                <span className="block opacity-75">Kart Numarası</span>
                <span className="font-mono font-semibold tracking-wider">{card?.id || 'Kart bilgisi yok'}</span>
              </div>
              <div>
                <span className="block opacity-75">Durum</span>
                <span className="badge badge-success text-xs">Aktif</span>
              </div>
            </div>
          </div>
        </div>

        {/* Quick Recharge & Turnstile */}
        <div className="md:col-span-2">
          <Card
            title="Bakiye Yükleme ve Deneme Binişi"
            subtitle="Kiosk simülasyonu üzerinden bakiye yükleyebilir veya deneme binişi yapabilirsiniz"
          >
            <div className="space-y-4">
              {cardDataError && <p role="alert" className="rounded-lg border border-warning/30 bg-surface-muted p-3 text-sm text-warning">{cardDataError}</p>}
              <div className="flex flex-wrap gap-3">
                <Button
                  variant="primary"
                  loading={rechargeLoading && selectedAmount === 50}
                  disabled={!card?.id || cardDataLoading}
                  onClick={() => onRecharge(50)}
                >
                  +50 TL Yükle
                </Button>
                <Button
                  variant="outline"
                  loading={rechargeLoading && selectedAmount === 100}
                  disabled={!card?.id || cardDataLoading}
                  onClick={() => onRecharge(100)}
                >
                  +100 TL Yükle
                </Button>
                <Button
                  variant="outline"
                  loading={rechargeLoading && selectedAmount === 200}
                  disabled={!card?.id || cardDataLoading}
                  onClick={() => onRecharge(200)}
                >
                  +200 TL Yükle
                </Button>
              </div>

              {/* 90-Minute Transfer Window Indicator */}
              <div className="transfer-window-box p-4 rounded-lg bg-surface-muted border border-border">
                <div className="flex justify-between items-center">
                  <div>
                    <h4 className="font-semibold text-sm">90 Dakika Aktarma Durumu</h4>
                    <p className="text-xs text-muted">
                      {transferTimer > 0 ? (
                        transferLevel === 1 ? (
                          <span className="text-success font-medium">
                            1. aktarma hakkı aktif (En fazla 2 aktarma yapılabilir). Farklı hatta indirimli geçiş uygulanır.
                          </span>
                        ) : transferLevel === 2 ? (
                          <span className="text-amber-500 font-medium">
                            2. aktarma hakkı aktif (Son aktarma hakkı). Farklı hatta indirimli geçiş uygulanır.
                          </span>
                        ) : (
                          <span className="text-danger font-medium">
                            Maksimum 2 aktarma sınırına ulaşıldı. 3. veya daha fazla aktarma seçeneği kesinlikle yoktur; yeni binişler tam tarifedir.
                          </span>
                        )
                      ) : (
                        'Aktif aktarma penceresi bulunmuyor. İlk biniş standart tarifeden ücretlendirilir.'
                      )}
                    </p>
                  </div>
                  {transferTimer > 0 && (
                    <div className="font-mono text-xl font-bold text-primary">
                      {formatTimer(transferTimer)}
                    </div>
                  )}
                </div>
              </div>

              {/* Turnstile Tap */}
              <div className="pt-2">
                <h4 className="text-sm font-semibold mb-2">Deneme Binişi:</h4>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={tapLoading || !card?.id || cardDataLoading}
                    onClick={() => onTapTurnstile('LINE_5', '5')}
                  >
                    {tapLoading ? '...' : '5 ESHOT'}
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={tapLoading || !card?.id || cardDataLoading}
                    onClick={() => onTapTurnstile('LINE_10', '10')}
                  >
                    {tapLoading ? '...' : '10 ESHOT'}
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={tapLoading || !card?.id || cardDataLoading}
                    onClick={() => onTapTurnstile('LINE_15', '15')}
                  >
                    {tapLoading ? '...' : '15 ESHOT'}
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={tapLoading || !card?.id || cardDataLoading}
                    onClick={() => onTapTurnstile('LINE_16', '16')}
                  >
                    {tapLoading ? '...' : '16 ESHOT'}
                  </Button>
                </div>
              </div>
            </div>
          </Card>
        </div>
      </div>

      {/* Card Ledger History Table with pagination */}
      <Card
        title="Hesap ve Biniş Hareketleri"
        subtitle="Son yapılan dolum, biniş ve aktarma işlemleri (Append-Only Finansal Defter)"
      >
        <Table
          columns={ledgerColumns}
          data={pagedLedger}
          keyExtractor={(row) => row.id || row.created_at}
          emptyMessage={cardDataLoading ? 'Kart hareketleri yükleniyor…' : cardDataError ? 'Kart hareketleri alınamadı.' : 'Henüz bir kart hareketi bulunmuyor.'}
          ariaLabel="Kart Hareketleri Geçmişi"
        />
        {totalPages > 1 && (
          <div className="flex items-center justify-between mt-4 text-sm">
            <span className="text-muted">
              {ledgerPage * PAGE_SIZE + 1}–{Math.min((ledgerPage + 1) * PAGE_SIZE, ledger.length)} / {ledger.length} kayıt
            </span>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={ledgerPage === 0}
                onClick={() => setLedgerPage((p) => Math.max(0, p - 1))}
              >
                ← Önceki
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={ledgerPage >= totalPages - 1}
                onClick={() => setLedgerPage((p) => Math.min(totalPages - 1, p + 1))}
              >
                Sonraki →
              </Button>
            </div>
          </div>
        )}
      </Card>
    </div>
  );
}
