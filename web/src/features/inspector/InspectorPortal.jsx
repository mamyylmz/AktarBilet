import React, { useState } from 'react';
import { Card } from '../../components/ui/Card.jsx';
import { Button } from '../../components/ui/Button.jsx';
import { Input } from '../../components/ui/Input.jsx';
import { Table } from '../../components/ui/Table.jsx';
import { formatKurus, formatDateTime } from '../../lib/formatters.js';

export function InspectorPortal({
  onQueryCard,
  queriedCard,
  queryLoading = false,
  queryError = null,
  onCreatePenalty,
  penaltyLoading = false,
  penalties = [],
}) {
  const [searchId, setSearchId] = useState('');
  const [penaltyCardId, setPenaltyCardId] = useState('');
  const [penaltyAmount, setPenaltyAmount] = useState('200'); // 200 TL default matching public tariff
  const [penaltyReason, setPenaltyReason] = useState('kartsiz_binis');

  const handleSearch = (e) => {
    if (e) e.preventDefault();
    if (searchId && typeof onQueryCard === 'function') {
      onQueryCard(searchId);
    }
  };

  const handleIssuePenalty = (e) => {
    if (e) e.preventDefault();
    const amountKurus = Number(penaltyAmount) * 100;
    if (typeof onCreatePenalty === 'function') {
      onCreatePenalty(penaltyCardId || searchId, amountKurus, penaltyReason);
    }
  };

  const penaltyColumns = [
    {
      header: 'Tarih / Saat',
      key: 'created_at',
      render: (row) => <span className="text-sm font-medium">{formatDateTime(row.created_at || row.issued_at)}</span>,
    },
    {
      header: 'Kart ID',
      key: 'card_id',
      render: (row) => <span className="font-mono">{row.card_id}</span>,
    },
    {
      header: 'Ceza Tutarı',
      key: 'penalty_amount_kurus',
      align: 'right',
      render: (row) => (
        <span className="font-mono font-semibold text-danger">
          {formatKurus(row.penalty_amount_kurus)}
        </span>
      ),
    },
    {
      header: 'İhlal Sebebi',
      key: 'reason',
      render: (row) => {
        const reasonLabels = {
          kartsiz_binis: 'Kartsız / Yetkisiz Biniş',
          baskasinin_karti: 'Başkasına Ait Kart',
          sahte_belge: 'Geçersiz İndirim Belgesi',
          usulsuz_kart_kullanimi: 'Usulsüz Kart Kullanımı',
          sahte_kart: 'Sahte / Değiştirilmiş Kart',
          diger: 'Diğer İhlal',
        };
        return (
          <span className="badge badge-danger">
            {reasonLabels[row.reason] || row.reason}
          </span>
        );
      },
    },
    {
      header: 'Tutanak Durumu',
      key: 'status',
      render: () => <span className="badge badge-success">Onaylandı</span>,
    },
  ];

  return (
    <div className="portal-container space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Card Search & Inspection */}
        <Card
          title="Saha Kart Doğrulama"
          subtitle="Yolcunun kart ID veya barkodunu sorgulayarak geçerlilik durumunu inceleyin"
        >
          <form onSubmit={handleSearch} className="space-y-4">
            <div className="flex gap-2 items-end">
              <div className="flex-1">
                <Input
                  id="inspector-card-search"
                  label="Sorgulanacak Kart ID"
                  value={searchId}
                  onChange={(e) => setSearchId(e.target.value)}
      placeholder="İncelenecek kart ID'si"
                  required
                />
              </div>
              <Button type="submit" variant="primary" loading={queryLoading}>
                Sorgula
              </Button>
            </div>

            {queryError && (
              <div role="alert" className="p-3 bg-surface-muted border border-danger/30 rounded-lg text-sm text-danger mt-3">
                {queryError}
              </div>
            )}

            {queriedCard && (
              <div className="card-result-box p-4 rounded-lg bg-surface-muted border border-border space-y-2 mt-4">
                <div className="flex justify-between items-center">
                  <span className="text-sm font-semibold">Kart Numarası:</span>
                  <span className="font-mono text-primary font-bold">{queriedCard.id || queriedCard.card_id}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-semibold">Kart Türü:</span>
                  <span className="badge badge-primary uppercase">{queriedCard.card_type || queriedCard.type || 'TAM'}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-semibold">Güncel Bakiye:</span>
                  <span className="font-mono text-lg font-bold text-success">
                    {queriedCard.balance_kurus !== undefined && queriedCard.balance_kurus !== null
                      ? formatKurus(queriedCard.balance_kurus)
                      : 'Bakiye bilgisi alınamadı'}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-semibold">Geçerlilik Durumu:</span>
                  <span className="badge badge-success">Aktif Kart</span>
                </div>
              </div>
            )}
          </form>
        </Card>

        {/* Issue Penalty Form */}
        <Card
          title="Ceza Tutanağı Düzenle"
          subtitle="Usulsüz kart kullanımı veya kaçak biniş durumunda resmi ceza kaydı oluşturun"
        >
          <form onSubmit={handleIssuePenalty} className="space-y-4">
              <Input
              id="penalty-card-id"
              label="İhlal Yapan Kart ID"
              value={penaltyCardId}
              onChange={(e) => setPenaltyCardId(e.target.value)}
                  placeholder="Örn: CARD_YOLCU_001"
              required
            />

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label htmlFor="penalty-amount-input" className="label text-sm font-semibold mb-1 block">
                  Ceza Tutarı (TL)
                </label>
                <input
                  id="penalty-amount-input"
                  type="number"
                  value={penaltyAmount}
                  onChange={(e) => setPenaltyAmount(e.target.value)}
                  className="input w-full font-mono font-semibold"
                  required
                />
              </div>

              <div>
                <label htmlFor="penalty-reason-select" className="label text-sm font-semibold mb-1 block">
                  İhlal Türü
                </label>
                  <select
                  id="penalty-reason-select"
                  value={penaltyReason}
                  onChange={(e) => setPenaltyReason(e.target.value)}
                  className="input w-full"
                >
                  <option value="kartsiz_binis">Kartsız / Yetkisiz Biniş</option>
                  <option value="baskasinin_karti">Başkasına Ait Kart Kullanımı</option>
                  <option value="sahte_belge">Geçersiz İndirim Belgesi</option>
                  <option value="usulsuz_kart_kullanimi">Usulsüz Kart Kullanımı</option>
                  <option value="sahte_kart">Sahte / Değiştirilmiş Kart</option>
                  <option value="diger">Diğer İhlal</option>
                </select>
              </div>
            </div>

            <Button
              type="submit"
              variant="danger"
              loading={penaltyLoading}
              className="w-full mt-2"
            >
              Ceza Tutanağı Düzenle
            </Button>
          </form>
        </Card>
      </div>

      {/* Penalties Issued History Table */}
      <Card
        title="Düzenlenen Saha Ceza Tutanakları"
        subtitle="Vardiyanızda sisteme kaydedilen resmi ihlal kayıtları ve denetim logları"
      >
        <Table
          columns={penaltyColumns}
          data={penalties}
          keyExtractor={(row) => row.id || row.created_at}
          emptyMessage="Bu vardiyada henüz bir ceza tutanağı düzenlenmemiştir."
          ariaLabel="Düzenlenen Ceza Tutanakları Tablosu"
        />
      </Card>
    </div>
  );
}
