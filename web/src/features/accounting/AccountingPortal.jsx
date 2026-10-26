import React, { useState, useId } from 'react';
import { Card } from '../../components/ui/Card.jsx';
import { Button } from '../../components/ui/Button.jsx';
import { Input } from '../../components/ui/Input.jsx';
import { Table } from '../../components/ui/Table.jsx';
import { formatKurus } from '../../lib/formatters.js';

export function AccountingPortal({
  isPeriodClosed,
  onClosePeriod,
  onReopenPeriod,
  onStorno,
  closingPeriod = false,
  stornoLoading = false,
  settlementData = null,
  loading = false,
}) {
  const [stornoRef, setStornoRef] = useState('REF_REC_500T_001');
  const [stornoReason, setStornoReason] = useState('Mükerrer çekim düzeltmesi');
  const [stornoAmount, setStornoAmount] = useState('250');
  const [reopenReason, setReopenReason] = useState('Yıllık finansal denetim ve mutabakat incelemesi');
  const [showReopenConfirm, setShowReopenConfirm] = useState(false);
  const [reopening, setReopening] = useState(false);
  const reopenReasonId = useId();

  const handleStornoSubmit = (e) => {
    if (e) e.preventDefault();
    const amountKurus = -Math.abs(Number(stornoAmount) * 100);
    onStorno(stornoRef, stornoReason, amountKurus);
  };

  const handleReopenSubmit = async (e) => {
    if (e) e.preventDefault();
    if (!onReopenPeriod) return;
    setReopening(true);
    try {
      await onReopenPeriod('2026-EKIM', reopenReason);
      setShowReopenConfirm(false);
    } finally {
      setReopening(false);
    }
  };

  // Extract or default dynamic operator shares from settlementData
  const shares = settlementData?.operator_shares || [
    {
      operator: 'İzmir Büyükşehir Belediyesi (ESHOT & Metro)',
      trips: 42580,
      share_kurus: 85160000,
      fee_kurus: 1703200,
      net_kurus: 83456800,
    },
    {
      operator: 'Özel Halk Otobüsleri Kooperatifi',
      trips: 18450,
      share_kurus: 36900000,
      fee_kurus: 738000,
      net_kurus: 36162000,
    },
    {
      operator: 'İZDENİZ Vapur İşletmeciliği',
      trips: 12400,
      share_kurus: 24800000,
      fee_kurus: 496000,
      net_kurus: 24304000,
    },
  ];

  // Calculate totals for Section 11.1
  const totals = shares.reduce(
    (acc, row) => ({
      trips: acc.trips + (Number(row.trips) || 0),
      share_kurus: acc.share_kurus + (Number(row.share_kurus) || 0),
      fee_kurus: acc.fee_kurus + (Number(row.fee_kurus) || 0),
      net_kurus: acc.net_kurus + (Number(row.net_kurus) || 0),
    }),
    { trips: 0, share_kurus: 0, fee_kurus: 0, net_kurus: 0 }
  );

  const shareColumns = [
    {
      header: 'İşletmeci Kurum',
      key: 'operator',
      render: (row) => (
        <span className={`text-sm ${row.is_total ? 'font-bold text-primary' : 'font-semibold'}`}>
          {row.operator}
        </span>
      ),
    },
    {
      header: 'Toplam Yolculuk',
      key: 'trips',
      align: 'right',
      render: (row) => (
        <span className={`font-mono ${row.is_total ? 'font-bold' : ''}`}>
          {Number(row.trips || 0).toLocaleString('tr-TR')}
        </span>
      ),
    },
    {
      header: 'Hak Ediş Tutarı',
      key: 'share_kurus',
      align: 'right',
      render: (row) => (
        <span className={`font-mono ${row.is_total ? 'font-bold text-primary text-base' : 'font-semibold text-primary'}`}>
          {formatKurus(row.share_kurus)}
        </span>
      ),
    },
    {
      header: 'Sistem Hizmet Bedeli (%2)',
      key: 'fee_kurus',
      align: 'right',
      render: (row) => (
        <span className={`font-mono text-muted ${row.is_total ? 'font-bold' : ''}`}>
          {formatKurus(row.fee_kurus)}
        </span>
      ),
    },
    {
      header: 'Net Ödenecek Pay',
      key: 'net_kurus',
      align: 'right',
      render: (row) => (
        <span className={`font-mono font-bold text-success ${row.is_total ? 'text-base underline' : ''}`}>
          {formatKurus(row.net_kurus)}
        </span>
      ),
    },
  ];

  // Append TOPLAM row (Section 11.1 requirement)
  const tableDataWithTotal = [
    ...shares,
    {
      operator: 'TOPLAM',
      trips: totals.trips,
      share_kurus: totals.share_kurus,
      fee_kurus: totals.fee_kurus,
      net_kurus: totals.net_kurus,
      is_total: true,
    },
  ];

  return (
    <div className="portal-container space-y-6">
      {/* Overview KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="kpi-card p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted font-medium block">Aktif Dönem</span>
          <span className="font-bold text-xl text-foreground">2026-EKIM</span>
          <span className={`badge ${isPeriodClosed ? 'badge-danger' : 'badge-success'} text-xs mt-2 block w-fit`}>
            {isPeriodClosed ? 'Kapalı Dönem' : 'Açık Dönem'}
          </span>
        </div>
        <div className="kpi-card p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted font-medium block">Toplam Biniş Geliri</span>
          <span className="font-mono font-bold text-xl text-primary">
            {formatKurus(totals.share_kurus || 146860000)}
          </span>
          <span className="text-xs text-muted block mt-1">
            {Number(totals.trips || 73430).toLocaleString('tr-TR')} Turnike Geçişi
          </span>
        </div>
        <div className="kpi-card p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted font-medium block">Toplam Dolum (Kiosk)</span>
          <span className="font-mono font-bold text-xl text-success">1.850.250,00 ₺</span>
          <span className="text-xs text-muted block mt-1">37.005 Dolum İşlemi</span>
        </div>
        <div className="kpi-card p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted font-medium block">Aktarma İndirim Desteği</span>
          <span className="font-mono font-bold text-xl text-foreground">184.200,00 ₺</span>
          <span className="text-xs text-muted block mt-1">Büyükşehir Sübvansiyonu</span>
        </div>
      </div>

      {/* Operator Revenue Shares Table */}
      <Card
        title="İşletmeci Gelir Paylaşımı ve Hak Ediş Tablosu"
        subtitle="Hat ve mod bazlı binişlerin işletmecilere sözleşme oranında dağılımı ve toplam mutabakat"
      >
        <Table
          columns={shareColumns}
          data={tableDataWithTotal}
          keyExtractor={(row) => row.operator}
          ariaLabel="İşletmeci Gelir Paylaşımı Tablosu"
          loading={loading}
        />
      </Card>

      {/* Actions: Period Close / Reopen & Storno */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Period Close & Reopen */}
        <Card
          title="Dönem Kapanış ve Denetim Onayı"
          subtitle="Aylık mutabakatı mühürleme veya yetkili denetim kapsamında yeniden açma"
        >
          <div className="space-y-4">
            <p className="text-sm text-foreground">
              Dönem kapatıldığında geçmişe dönük biniş defteri dondurulur, işletmeci hakediş talimatları kesinleşir.
            </p>

            {isPeriodClosed ? (
              <div className="space-y-3">
                <div role="status" className="p-3 bg-surface-muted border border-danger/30 rounded-lg text-sm text-danger font-medium">
                  🔒 Bu dönem (2026-EKIM) başarıyla kapatılmıştır. Kayıtlar kilitli ve denetime hazırdır.
                </div>

                {/* Section 11.4 Reopen 2026-EKIM Action */}
                {!showReopenConfirm ? (
                  <Button
                    variant="outline"
                    onClick={() => setShowReopenConfirm(true)}
                    className="w-full"
                  >
                    2026-EKIM Dönemini Tekrar Aç (Reopen)
                  </Button>
                ) : (
                  <form onSubmit={handleReopenSubmit} className="p-3 bg-surface-muted border border-border rounded-lg space-y-3">
                    <p className="text-xs font-semibold text-warning">
                      ⚠️ Yetkili Denetim İşlemi: 2026-EKIM dönemi mutabakat düzeltmeleri için yeniden açılacaktır. Bu eylem audit loguna kaydedilir.
                    </p>
                    <div>
                      <label htmlFor={reopenReasonId} className="text-xs font-medium block mb-1">
                        Yeniden Açma Gerekçesi
                      </label>
                      <input
                        id={reopenReasonId}
                        type="text"
                        value={reopenReason}
                        onChange={(e) => setReopenReason(e.target.value)}
                        className="input text-xs w-full"
                        required
                      />
                    </div>
                    <div className="flex gap-2">
                      <Button
                        type="submit"
                        variant="danger"
                        size="sm"
                        loading={reopening}
                        className="flex-1"
                      >
                        Onayla ve Aç
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => setShowReopenConfirm(false)}
                      >
                        İptal
                      </Button>
                    </div>
                  </form>
                )}
              </div>
            ) : (
              <Button
                variant="danger"
                onClick={onClosePeriod}
                loading={closingPeriod}
                className="w-full"
              >
                Dönemi Kapat
              </Button>
            )}
          </div>
        </Card>

        {/* Storno Correction */}
        <Card
          title="Storno ve Hatalı Kayıt Düzeltme"
          subtitle="Hatalı veya mükerrer turnike işlemlerine resmi ters kayıt (storno) düzenleme"
        >
          <form onSubmit={handleStornoSubmit} className="space-y-4">
            <Input
              id="storno-ref-id"
              label="İşlem Referans Kodu (Reference ID)"
              value={stornoRef}
              onChange={(e) => setStornoRef(e.target.value)}
              placeholder="Örn: REF_REC_500T_001"
              required
            />
            <Input
              id="storno-amount-input"
              label="Storno Tutarı (TL)"
              type="number"
              value={stornoAmount}
              onChange={(e) => setStornoAmount(e.target.value)}
              placeholder="Örn: 250"
              required
            />
            <Input
              id="storno-reason-input"
              label="Storno Gerekçesi"
              value={stornoReason}
              onChange={(e) => setStornoReason(e.target.value)}
              placeholder="Örn: Mükerrer biniş iadesi"
              required
            />
            <Button
              type="submit"
              variant="outline"
              loading={stornoLoading}
              className="w-full"
            >
              Storno Düzeltme
            </Button>
          </form>
        </Card>
      </div>
    </div>
  );
}
