import React, { useState } from 'react';
import { Card } from '../../components/ui/Card.jsx';
import { Button } from '../../components/ui/Button.jsx';
import { Table } from '../../components/ui/Table.jsx';
import { formatKurus } from '../../lib/formatters.js';

export function LineManagerPortal({
  lineFinancials = [],
  onSimulateScenario,
  onApproveScenario,
  optimizationResult,
  loadingFinancials = false,
  simulating = false,
  approving = false,
}) {
  const [selectedLine, setSelectedLine] = useState('');
  const [currentHeadway, setCurrentHeadway] = useState('15');
  const [targetHeadway, setTargetHeadway] = useState('10');
  const [fleetCapacity, setFleetCapacity] = useState('100');
  const [peakHour, setPeakHour] = useState(true);
  const [approvalNotes, setApprovalNotes] = useState('Pik saatler sefer artırımı ve frekans optimizasyonu onayı');

  const handleSimulate = (e) => {
    if (e) e.preventDefault();
    if (typeof onSimulateScenario === 'function') {
      onSimulateScenario({
        line_id: selectedLine,
        current_headway_minutes: Number(currentHeadway) || 15,
        target_headway_minutes: Number(targetHeadway) || 10,
        fleet_capacity_per_vehicle: Number(fleetCapacity) || 100,
        peak_hour: peakHour,
      });
    }
  };

  const handleApprove = () => {
    if (!optimizationResult || typeof onApproveScenario !== 'function') return;
    onApproveScenario({
      line_id: selectedLine,
      current_headway_minutes: Number(currentHeadway) || 15,
      target_headway_minutes: Number(targetHeadway) || 10,
      fleet_capacity: Number(fleetCapacity) || 100,
      vehicle_delta: optimizationResult.vehicle_delta || 0,
      delta_cost_kurus: optimizationResult.delta_cost_kurus || 0,
      projected_net_margin_kurus: optimizationResult.projected_net_margin_kurus || 0,
      notes: approvalNotes,
    });
  };

  const financialColumns = [
    {
      header: 'Hat',
      key: 'line_name',
      minWidth: '220px',
      render: (row) => <span className="font-semibold text-sm">{row.line_name || row.line_id}</span>,
    },
    {
      header: 'İşletmeci',
      key: 'operator',
      minWidth: '110px',
      render: (row) => (
        <span className="badge badge-secondary uppercase">{row.operator}</span>
      ),
    },
    {
      header: 'Toplam Biniş',
      key: 'total_boardings',
      minWidth: '120px',
      align: 'right',
      render: (row) => <span className="font-mono">{Number(row.total_boardings || 0).toLocaleString('tr-TR')}</span>,
    },
    {
      header: 'Brüt Gelir',
      key: 'gross_revenue_kurus',
      minWidth: '130px',
      align: 'right',
      render: (row) => <span className="font-mono text-primary font-semibold">{formatKurus(row.gross_revenue_kurus)}</span>,
    },
    {
      header: 'Toplam Maliyet',
      key: 'total_cost_kurus',
      minWidth: '140px',
      align: 'right',
      render: (row) => <span className="font-mono text-muted">{formatKurus(row.total_cost_kurus)}</span>,
    },
    {
      header: 'Net Kâr / Marj',
      key: 'net_margin_kurus',
      minWidth: '140px',
      align: 'right',
      render: (row) => {
        const isProfitable = (row.net_margin_kurus || 0) >= 0;
        return (
          <span className={`font-mono font-bold ${isProfitable ? 'text-success' : 'text-danger'}`}>
            {formatKurus(row.net_margin_kurus)}
          </span>
        );
      },
    },
  ];

  return (
    <div className="portal-container space-y-6">
      {/* Line Financial Performance */}
      <Card
        title="Hat Maliyet ve Gelir Raporları"
        subtitle="Hat bazlı yolcu binişleri, operasyonel yakıt ve personel giderleri, net kârlılık marjı"
      >
        <Table
          columns={financialColumns}
          data={lineFinancials}
          keyExtractor={(row) => row.line_id || row.id}
          loading={loadingFinancials}
          emptyMessage="Kayıtlı hat maliyet verisi bulunamadı."
          ariaLabel="Hat Maliyet ve Kârlılık Tablosu"
        />
      </Card>

      {/* Headway Simulation & Optimization */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Simulation Parameters Form */}
        <Card
          title="Sefer Sıklığı (Headway) Optimizasyon Simülatörü"
          subtitle="Hat frekans ve kapasite parametrelerini girerek maliyet-fayda analizini çalıştırın"
        >
          <form onSubmit={handleSimulate} className="space-y-4">
            <div>
              <label htmlFor="sim-line-select" className="label text-sm font-semibold mb-1 block">
                İncelenecek Hat
              </label>
              <select
                id="sim-line-select"
                value={selectedLine || lineFinancials[0]?.line_id || ''}
                onChange={(e) => setSelectedLine(e.target.value)}
                className="input w-full"
              >
                {lineFinancials.map((line) => (
                  <option key={line.line_id} value={line.line_id}>{line.line_name || line.line_id}</option>
                ))}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="sim-curr-headway" className="label text-sm font-semibold mb-1 block">
                  Mevcut Sıklık (dk)
                </label>
                <input
                  id="sim-curr-headway"
                  type="number"
                  min="1"
                  max="180"
                  value={currentHeadway}
                  onChange={(e) => setCurrentHeadway(e.target.value)}
                  className="input w-full font-mono"
                  required
                />
              </div>

              <div>
                <label htmlFor="sim-target-headway" className="label text-sm font-semibold mb-1 block">
                  Hedef Sıklık (dk)
                </label>
                <input
                  id="sim-target-headway"
                  type="number"
                  min="1"
                  max="180"
                  value={targetHeadway}
                  onChange={(e) => setTargetHeadway(e.target.value)}
                  className="input w-full font-mono"
                  required
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label htmlFor="sim-fleet-cap" className="label text-sm font-semibold mb-1 block">
                  Araç Kapasitesi
                </label>
                <input
                  id="sim-fleet-cap"
                  type="number"
                  min="20"
                  max="1000"
                  value={fleetCapacity}
                  onChange={(e) => setFleetCapacity(e.target.value)}
                  className="input w-full font-mono"
                  required
                />
              </div>

              <div className="flex items-center pt-6">
                <label className="flex items-center gap-2 cursor-pointer text-sm">
                  <input
                    type="checkbox"
                    checked={peakHour}
                    onChange={(e) => setPeakHour(e.target.checked)}
                    className="checkbox"
                  />
                  <span className="font-semibold">Pik Saat Koşulları</span>
                </label>
              </div>
            </div>

            <Button
              type="submit"
              variant="primary"
              loading={simulating}
              className="w-full"
            >
              Senaryoyu Simüle Et
            </Button>
          </form>
        </Card>

        {/* Simulation Output and Approval Decision */}
        <Card
          title="Senaryo Analizi ve Karar Onayı"
          subtitle="Model çıktısı, ek filo ihtiyacı ve net marj projeksiyonu"
        >
          {optimizationResult ? (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div className="p-3 bg-surface-muted rounded-lg border border-border">
                  <span className="text-xs text-muted block">Araç Filo Değişimi:</span>
                  <span className="font-mono text-xl font-bold text-primary">
                    {optimizationResult.vehicle_delta > 0 ? `+${optimizationResult.vehicle_delta}` : optimizationResult.vehicle_delta} Araç
                  </span>
                  <span className="text-xs text-muted block mt-1">
                    {optimizationResult.current_vehicles} → {optimizationResult.proposed_vehicles} Araç
                  </span>
                </div>

                <div className="p-3 bg-surface-muted rounded-lg border border-border">
                  <span className="text-xs text-muted block">Öngörülen Net Marj:</span>
                  <span className={`font-mono text-xl font-bold ${optimizationResult.projected_net_margin_kurus >= 0 ? 'text-success' : 'text-danger'}`}>
                    {formatKurus(optimizationResult.projected_net_margin_kurus)}
                  </span>
                  <span className="text-xs text-muted block mt-1">
                    Ek Maliyet: {formatKurus(optimizationResult.delta_cost_kurus)}
                  </span>
                </div>
              </div>

              <div className="p-3 bg-surface-muted rounded-lg border border-border text-sm">
                <span className="font-semibold block mb-1">Sistem Önerisi:</span>
                <p className="text-foreground">{optimizationResult.recommended_action || 'Senaryo uygulanabilir.'}</p>
              </div>

              <div>
                <label htmlFor="sim-approval-notes" className="label text-sm font-semibold mb-1 block">
                  Yönetici Onay Notu:
                </label>
                <input
                  id="sim-approval-notes"
                  type="text"
                  value={approvalNotes}
                  onChange={(e) => setApprovalNotes(e.target.value)}
                  className="input w-full text-sm"
                  placeholder="Onay gerekçesini yazınız"
                />
              </div>

              <Button
                variant="primary"
                onClick={handleApprove}
                loading={approving}
                className="w-full"
              >
                Duran Senaryoyu Onayla (Persist)
              </Button>
            </div>
          ) : (
            <div className="text-center py-12 text-muted">
              <p className="text-sm">Parametreleri belirleyip "Senaryoyu Simüle Et" butonuna tıklayarak model analizini başlatın.</p>
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}
