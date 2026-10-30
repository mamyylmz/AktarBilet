import React, { useState, useId, useRef } from 'react';
import { Card } from '../../components/ui/Card.jsx';
import { Button } from '../../components/ui/Button.jsx';
import { Input } from '../../components/ui/Input.jsx';
import { Table } from '../../components/ui/Table.jsx';
import { formatKurus } from '../../lib/formatters.js';
import { formatDateTime } from '../../lib/formatters.js';

export function AccountingPortal({
  isPeriodClosed,
  onClosePeriod,
  onReopenPeriod,
  onStorno,
  onAddSubsidy,
  addingSubsidy = false,
  closingPeriod = false,
  stornoLoading = false,
  settlementData = null,
  loading = false,
  token,
  onFetchSettlements,
  onLoginAsMuhasebe,
}) {
  const [stornoRef, setStornoRef] = useState('REF_REC_500T_001');
  const [stornoReason, setStornoReason] = useState('Mükerrer çekim düzeltmesi');
  const [stornoAmount, setStornoAmount] = useState('250');
  const [reopenReason, setReopenReason] = useState('Yıllık finansal denetim ve mutabakat incelemesi');
  const [showReopenConfirm, setShowReopenConfirm] = useState(false);
  const [reopening, setReopening] = useState(false);
  const [subsidyOperator, setSubsidyOperator] = useState('belediye');
  const [subsidyAmountKurus, setSubsidyAmountKurus] = useState('');
  const [subsidyReason, setSubsidyReason] = useState('');
  const [submittingSubsidy, setSubmittingSubsidy] = useState(false);
  const subsidyKeyRef = useRef(null);
  const reopenReasonId = useId();
  const subsidyOperatorId = useId();
  const subsidyAmountId = useId();
  const subsidyReasonId = useId();

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
      await onReopenPeriod(periodName, reopenReason);
      setShowReopenConfirm(false);
    } finally {
      setReopening(false);
    }
  };

  const handleSubsidySubmit = async (event) => {
    event.preventDefault();
    const amountKurus = Number(subsidyAmountKurus);
    if (!Number.isSafeInteger(amountKurus) || amountKurus <= 0 || !settlementData?.is_closed || !onAddSubsidy) return;
    const payload = { period_name: periodName, operator: subsidyOperator, amount_kurus: amountKurus, reason: subsidyReason.trim() };
    const signature = JSON.stringify(payload);
    if (subsidyKeyRef.current?.signature !== signature) {
      subsidyKeyRef.current = { signature, key: `SUBSIDY_${globalThis.crypto?.randomUUID?.() || `${Date.now()}_${Math.random().toString(16).slice(2)}`}` };
    }
    setSubmittingSubsidy(true);
    try {
      const created = await onAddSubsidy(payload, subsidyKeyRef.current.key);
      if (created) {
        setSubsidyAmountKurus('');
        setSubsidyReason('');
        subsidyKeyRef.current = null;
      }
    } finally {
      setSubmittingSubsidy(false);
    }
  };

  // Extract or default dynamic operator shares from settlementData
  const shares = settlementData?.operator_shares || [];
  const periodName = settlementData?.period_name || 'Dönem verisi bekleniyor';

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
      {/* Oturum Açılmadı / Yetki Uyarısı */}
      {!token && (
        <div className="p-4 bg-amber-500/10 border border-amber-500/30 rounded-lg flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <span className="text-xl" aria-hidden="true">⚠️</span>
            <div>
              <h4 className="text-sm font-semibold text-foreground">Oturum Açılmadı (Muhasebe Yetkisi Gerekli)</h4>
              <p className="text-xs text-muted">
                Dönem mutabakatını kesinleştirmek veya denetim amaçlı yeniden açmak için sisteme Muhasebe veya Admin rolüyle giriş yapmalısınız.
              </p>
            </div>
          </div>
          {onLoginAsMuhasebe && (
            <Button
              variant="primary"
              size="sm"
              onClick={onLoginAsMuhasebe}
              className="whitespace-nowrap font-medium"
            >
              🔐 Muhasebe Girişi Yap
            </Button>
          )}
        </div>
      )}

      {/* Overview KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="kpi-card p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted font-medium block">Aktif Dönem</span>
          <span className="font-bold text-xl text-foreground">{periodName}</span>
          <span className={`badge ${isPeriodClosed ? 'badge-danger' : 'badge-success'} text-xs mt-2 block w-fit`}>
            {isPeriodClosed ? 'Kapalı Dönem' : 'Açık Dönem'}
          </span>
        </div>
        <div className="kpi-card p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted font-medium block">Toplam Biniş Geliri</span>
          <span className="font-mono font-bold text-xl text-primary">
            {settlementData ? formatKurus(settlementData.total_revenue_kurus ?? totals.share_kurus) : '—'}
          </span>
          <span className="text-xs text-muted block mt-1">
            {settlementData ? Number(settlementData.total_trips ?? totals.trips).toLocaleString('tr-TR') : '—'} Turnike Geçişi
          </span>
        </div>
        <div className="kpi-card p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted font-medium block">Toplam Dolum (Kiosk)</span>
          <span className="font-mono font-bold text-xl text-success">{settlementData?.total_recharge_kurus != null ? formatKurus(settlementData.total_recharge_kurus) : '—'}</span>
          <span className="text-xs text-muted block mt-1">{settlementData?.total_recharge_count != null ? `${Number(settlementData.total_recharge_count).toLocaleString('tr-TR')} dolum işlemi` : 'Dolum verisi bulunmuyor'}</span>
        </div>
        <div className="kpi-card p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted font-medium block">Aktarma İndirim Desteği</span>
          <span className="font-mono font-bold text-xl text-foreground">
            {Number(settlementData?.total_subsidy_count) > 0 ? formatKurus(settlementData.total_subsidy_kurus) : '—'}
          </span>
          <span className="text-xs text-muted block mt-1">
            {settlementData?.total_subsidy_count == null ? 'Dönem verisi bekleniyor' : Number(settlementData.total_subsidy_count) > 0 ? `${Number(settlementData.total_subsidy_count).toLocaleString('tr-TR')} kayıt` : 'Bu dönem için destek kaydı yok'}
          </span>
        </div>
      </div>

      {/* Operator Revenue Shares Table */}
      <Card
        title="İşletmeci Gelir Paylaşımı ve Hak Ediş Tablosu"
        subtitle={`${periodName} döneminde kalıcı biniş olaylarından ve signed düzeltmelerden hesaplanır`}
      >
        <Table
          columns={shareColumns}
          data={tableDataWithTotal}
          keyExtractor={(row) => row.operator}
          ariaLabel="İşletmeci Gelir Paylaşımı Tablosu"
          loading={loading}
        />
      </Card>

      <Card title="Dönem ve Settlement Adjustment Denetim Geçmişi" subtitle="Kapanış, yeniden açma, storno ve destek hareketleri gerçek audit kayıtlarından gösterilir">
        {settlementData?.history?.length ? (
          <div className="table-responsive settlement-history-scroll" tabIndex={0} role="region" aria-label="Dönem ve settlement adjustment denetim geçmişi">
            <table className="settlement-history-table w-full text-sm">
              <thead><tr><th className="text-left p-3">İşlem</th><th className="text-left p-3">Dönem</th><th className="text-left p-3">Tarih / Saat</th><th className="text-left p-3">Gerekçe</th><th className="text-right p-3">Tutar</th><th className="text-left p-3">Kullanıcı</th><th className="text-left p-3">Audit durumu</th></tr></thead>
              <tbody>{settlementData.history.map((entry, index) => <tr key={`${entry.action}-${entry.created_at}-${index}`} className="border-t border-border">
                <td className="p-3">{entry.action === 'SETTLEMENT_CLOSE' ? 'Dönem kapatma' : entry.action === 'SETTLEMENT_REOPEN' ? 'Dönem yeniden açma' : entry.action === 'SETTLEMENT_SUBSIDY_ADJUSTMENT' ? 'Aktarma desteği' : 'Storno'}</td>
                <td className="p-3">{periodName}</td>
                <td className="p-3 whitespace-nowrap">{entry.created_at ? formatDateTime(entry.created_at) : 'Kayıt zamanı bulunamadı'}</td>
                <td className="settlement-history-reason p-3">{entry.reason || '—'}</td>
                <td className="p-3 text-right whitespace-nowrap">{entry.amount_kurus !== '' && entry.amount_kurus != null ? formatKurus(Number(entry.amount_kurus)) : '—'}</td>
                <td className="p-3">{entry.actor || '—'}</td>
                <td className="p-3"><span className="badge badge-success">Audit kaydedildi</span></td>
              </tr>)}</tbody>
            </table>
          </div>
        ) : <p className="text-sm text-muted">{loading ? 'Denetim geçmişi yükleniyor…' : 'Bu dönem için kayıtlı kapanış, yeniden açma veya storno hareketi yok.'}</p>}
      </Card>

      <Card title="Aktarma desteği settlement adjustment" subtitle="Yalnız kapatılmış döneme gerçek tutar ve gerekçe ile eklenir; kayıt ve audit aynı transaction içindedir.">
        {!isPeriodClosed && <p className="mb-4 text-sm text-muted">Adjustment eklemek için önce dönemi kapatın. Henüz kayıt yoksa rapor tutarı boş kalır.</p>}
        <form onSubmit={handleSubsidySubmit} className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label htmlFor={subsidyOperatorId} className="input-label">İşletmeci</label>
            <select id={subsidyOperatorId} className="input-field" value={subsidyOperator} onChange={(event) => setSubsidyOperator(event.target.value)}>
              <option value="belediye">İzmir Büyükşehir Belediyesi (ESHOT & Metro)</option>
              <option value="ozel_halk">Özel Halk Otobüsleri Kooperatifi</option>
              <option value="izdeniz">İZDENİZ Vapur İşletmeciliği</option>
            </select>
          </div>
          <div>
            <label htmlFor={subsidyAmountId} className="input-label">Tutar (tam sayı kuruş)</label>
            <input id={subsidyAmountId} className="input-field" type="number" min="1" step="1" inputMode="numeric" required value={subsidyAmountKurus} onChange={(event) => setSubsidyAmountKurus(event.target.value)} />
          </div>
          <div className="md:col-span-2">
            <label htmlFor={subsidyReasonId} className="input-label">Destek gerekçesi</label>
            <input id={subsidyReasonId} className="input-field" type="text" maxLength="500" required value={subsidyReason} onChange={(event) => setSubsidyReason(event.target.value)} />
          </div>
          <div className="md:col-span-2 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <p className="text-xs text-muted">Dönem: {periodName}. Bu işlem kart bakiyesini değiştirmez.</p>
            <Button type="submit" variant="primary" loading={submittingSubsidy || addingSubsidy} disabled={!isPeriodClosed || !settlementData || submittingSubsidy || addingSubsidy}>Adjustment kaydet</Button>
          </div>
        </form>
      </Card>

      {/* Actions: Period Close / Reopen & Storno */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Period Close & Reopen */}
        <Card
          title="Dönem Kapanış ve Denetim Onayı"
          subtitle="Aylık mutabakatı mühürleme veya yetkili denetim kapsamında yeniden açma"
        >
          <div className="space-y-4">
            {/* 1. Canlı Durum Rozeti (Status Badge) */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between p-3 rounded-lg border bg-surface-muted border-border gap-2">
              <div className="flex items-center gap-3">
                <span className="text-xs font-semibold text-foreground uppercase tracking-wider">
                  Aktif Dönem: <strong>{periodName}</strong>
                </span>
                {onFetchSettlements && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={onFetchSettlements}
                    loading={loading}
                    className="text-xs py-0.5 px-2 h-auto"
                    title="Dönem mutabakat durumunu sunucudan yenile"
                  >
                    🔄 Durumu Yenile
                  </Button>
                )}
              </div>
              {isPeriodClosed ? (
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold bg-danger/15 text-danger border border-danger/30">
                  <span className="w-2 h-2 rounded-full bg-danger animate-pulse" aria-hidden="true" />
                  🔒 DÖNEM DURUMU: KAPALI (Kayıtlar Kilitli & Denetime Hazır)
                </span>
              ) : (
                <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-bold bg-success/15 text-success border border-success/30">
                  <span className="w-2 h-2 rounded-full bg-success animate-pulse" aria-hidden="true" />
                  ● DÖNEM DURUMU: AÇIK (İşlemler Devam Ediyor)
                </span>
              )}
            </div>

            <p className="text-sm text-foreground">
              Dönem kapatıldığında geçmişe dönük biniş defteri dondurulur, işletmeci hakediş talimatları kesinleşir.
            </p>

            {/* 2. Akıllı Aksiyon Alanı */}
            {isPeriodClosed ? (
              <div className="space-y-3">
                <div role="status" className="p-3 bg-surface-muted border border-danger/30 rounded-lg text-sm text-danger font-medium">
                  🔒 Bu dönem ({periodName}) kapalıdır. Kayıtlar kilitli ve denetime hazırdır.
                </div>

                {/* Section 11.4 Reopen 2026-EKIM Action */}
                {!showReopenConfirm ? (
                  <Button
                    variant="outline"
                    onClick={() => setShowReopenConfirm(true)}
                    className="w-full font-medium"
                  >
                    {periodName} Dönemini Tekrar Aç (Reopen)
                  </Button>
                ) : (
                  <form onSubmit={handleReopenSubmit} className="p-3 bg-surface-muted border border-border rounded-lg space-y-3">
                    <p className="text-xs font-semibold text-warning">
                      ⚠️ Yetkili denetim işlemi: {periodName} dönemi mutabakat düzeltmeleri için yeniden açılacaktır. Bu eylem audit loguna kaydedilir.
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
                        placeholder="Örn: Denetim ve mutabakat düzeltmesi"
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
              <div className="space-y-2">
                <Button
                  variant="danger"
                  onClick={onClosePeriod}
                  loading={closingPeriod}
                  disabled={!settlementData}
                  className="w-full font-semibold"
                >
                  Dönemi Kapat ve Mutabakatı Kesinleştir
                </Button>
                <p className="text-xs text-muted text-center italic">
                  Dönem henüz kapatılmamıştır. Yeniden açma (reopen) işlemi yalnızca kapatılmış dönemlerde yetkili denetim amacıyla yapılabilir.
                </p>
              </div>
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
