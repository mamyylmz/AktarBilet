import React, { useState } from 'react';
import { Card } from '../../components/ui/Card.jsx';
import { Button } from '../../components/ui/Button.jsx';
import { Table } from '../../components/ui/Table.jsx';
import { formatKurus, formatDateTime } from '../../lib/formatters.js';

export function PassengerPortal({
  card,
  ledger = [],
  onRecharge,
  onTapTurnstile,
  rechargeLoading = false,
  transferTimer = 0,
  transferLevel = 0,
}) {
  const [selectedAmount, setSelectedAmount] = useState(50);
  const [customAmount, setCustomAmount] = useState('');

  const formatTimer = (secs) => {
    const mins = Math.floor(secs / 60);
    const s = secs % 60;
    return `${mins.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  };

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
        if (row.entry_type === 'dolum') {
          return <span className="badge badge-success">Bakiye Yükleme</span>;
        }
        if (row.entry_type === 'ceza') {
          return <span className="badge badge-danger">Ceza Kesintisi</span>;
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
      render: (row) => <span className="text-sm text-foreground">{row.description || 'Toplu Taşıma Geçişi'}</span>,
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
              <span className="card-type-badge font-mono uppercase">{card?.card_type || 'TAM'} KART</span>
            </div>
            <div className="card-middle-row my-4">
              <span className="card-balance-label text-xs uppercase tracking-wider text-white/80">Güncel Bakiye</span>
              <div className="card-balance-value font-mono text-3xl font-bold text-white mt-1">
                {formatKurus(card?.balance_kurus || 0)}
              </div>
            </div>
            <div className="card-bottom-row flex justify-between items-end text-xs text-white/90">
              <div>
                <span className="block opacity-75">Kart Numarası</span>
                <span className="font-mono font-semibold tracking-wider">{card?.id || 'CARD_YOLCU_001'}</span>
              </div>
              <div>
                <span className="block opacity-75">Durum</span>
                <span className="badge badge-success text-xs">Aktif</span>
              </div>
            </div>
          </div>
        </div>

        {/* Quick Recharge & Actions */}
        <div className="md:col-span-2">
          <Card
            title="Bakiye Yükleme (Demo Kiosk)"
            subtitle="Kredi kartı veya kiosk üzerinden güvenli bakiye yükleme simülasyonu"
          >
            <div className="space-y-4">
              <div className="flex flex-wrap gap-3">
                <Button
                  variant="primary"
                  loading={rechargeLoading && selectedAmount === 50}
                  onClick={() => onRecharge(50)}
                >
                  +50 TL Yükle
                </Button>
                <Button
                  variant="outline"
                  loading={rechargeLoading && selectedAmount === 100}
                  onClick={() => onRecharge(100)}
                >
                  +100 TL Yükle
                </Button>
                <Button
                  variant="outline"
                  loading={rechargeLoading && selectedAmount === 200}
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

              {/* Turnstile Tap Demonstration */}
              <div className="turnstile-demo-section pt-2">
                <h4 className="text-sm font-semibold mb-2">Turnike Biniş Simülasyonu:</h4>
                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => onTapTurnstile('LINE_500T', '500T')}
                  >
                    500T Otobüs
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => onTapTurnstile('LINE_14ES', '14ES')}
                  >
                    14ES Kadıköy
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => onTapTurnstile('LINE_T1', 'T1')}
                  >
                    T1 Tramvay
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => onTapTurnstile('LINE_VAPUR_KE', 'Vapur')}
                  >
                    Kadıköy Vapur
                  </Button>
                </div>
              </div>
            </div>
          </Card>
        </div>
      </div>

      {/* Card Ledger History Table */}
      <Card
        title="Hesap ve Biniş Hareketleri"
        subtitle="Son yapılan dolum, biniş ve aktarma işlemleri (Append-Only Finansal Defter)"
      >
        <Table
          columns={ledgerColumns}
          data={ledger}
          keyExtractor={(row) => row.id || row.created_at}
          emptyMessage="Henüz bir kart hareketi bulunmuyor."
          ariaLabel="Kart Hareketleri Geçmişi"
        />
      </Card>
    </div>
  );
}

