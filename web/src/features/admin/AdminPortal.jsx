import React, { useState, useEffect } from 'react';
import { Card } from '../../components/ui/Card';
import { Button } from '../../components/ui/Button';
import { Table } from '../../components/ui/Table';
import { api } from '../../services/api';
import { formatDate, formatDateTime } from '../../lib/formatters';

export function AdminPortal({ token, onShowToast, onRolePreview, activeRole }) {
  const [healthStatus, setHealthStatus] = useState(null);
  const [healthLatency, setHealthLatency] = useState(null);
  const [lastHealthCheck, setLastHealthCheck] = useState(null);
  const [loadingHealth, setLoadingHealth] = useState(false);
  const [anomalies, setAnomalies] = useState([]);
  const [loadingAnomalies, setLoadingAnomalies] = useState(false);
  const [anomaliesError, setAnomaliesError] = useState('');
  const [systemUsers, setSystemUsers] = useState([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
  const [usersError, setUsersError] = useState('');

  const fetchHealth = async () => {
    setLoadingHealth(true);
    const start = performance.now();
    try {
      const res = await api.getHealth();
      const latency = Math.round(performance.now() - start);
      if (!res || !['UP', 'DOWN'].includes(res.status)) throw new Error('Sağlık yanıtı doğrulanamadı.');
      setHealthLatency(latency);
      setLastHealthCheck(new Date().toISOString());
      setHealthStatus(res);
      if (onShowToast) onShowToast(res.status === 'UP' ? `API sağlık yanıtı alındı (${latency} ms)` : 'API sağlık kontrolü erişilemez durumu döndürdü.', res.status === 'UP' ? 'success' : 'error');
    } catch (err) {
      const latency = Math.round(performance.now() - start);
      setHealthLatency(latency);
      setLastHealthCheck(new Date().toISOString());
      setHealthStatus({ status: 'DOWN', error: err.message });
      if (onShowToast) onShowToast(`Sağlık kontrolü hatası: ${err.message}`, 'error');
    } finally {
      setLoadingHealth(false);
    }
  };

  const fetchAnomalies = async () => {
    setLoadingAnomalies(true);
    setAnomaliesError('');
    try {
      const res = await api.getAnomalies(token);
      setAnomalies(Array.isArray(res) ? res : res?.anomalies || res?.items || []);
    } catch (err) {
      setAnomalies([]);
      setAnomaliesError(err.message || 'Denetim günlüğü yüklenemedi.');
    } finally {
      setLoadingAnomalies(false);
    }
  };

  const fetchUsers = async () => {
    setLoadingUsers(true);
    setUsersError('');
    try {
      const users = await api.getAdminUsers(token);
      setSystemUsers(Array.isArray(users) ? users : []);
    } catch (err) {
      setSystemUsers([]);
      setUsersError(err.message || 'Kullanıcı dizini yüklenemedi.');
    } finally {
      setLoadingUsers(false);
    }
  };

  useEffect(() => {
    fetchHealth();
    fetchAnomalies();
    fetchUsers();
  }, [token]);

  return (
    <div className="portal-container admin-portal space-y-6" data-testid="admin-portal">
      {/* Section 4.2: Admin Only Role Preview Bar */}
      <Card
        title="Admin Rol Paneli Önizleme Modu"
        subtitle="Yalnızca sistem yöneticisi için yetkili portal arayüzlerini doğrudan test ve teftiş etme aracı"
      >
        <label className="flex flex-col sm:flex-row sm:items-center gap-2 text-sm">
          <span className="font-medium">Portal önizlemesi</span>
          <select aria-label="Önizleme portalı seç" value={activeRole} onChange={(event) => onRolePreview?.(event.target.value)} className="input w-full sm:w-auto">
            {[
              ['admin', 'Sistem Yönetimi'], ['yolcu', 'Yolcu'], ['kontrolor', 'Kontrolör'],
              ['hat_yoneticisi', 'Hat Yöneticisi'], ['muhasebe', 'Muhasebe'], ['operasyon', 'Operasyon'],
            ].map(([id, label]) => <option key={id} value={id}>{label}</option>)}
          </select>
          <span className="text-xs text-muted">Yalnızca arayüz önizlemesidir; yetki vermez.</span>
        </label>
      </Card>

      <div className="p-4 bg-surface-muted border border-border rounded-lg">
        <div>
          <h4 className="text-sm font-semibold text-foreground">Mali mutabakat durumu</h4>
          <p className="text-xs text-muted">Dönem kapama, yeniden açma ve storno geçmişi Muhasebe portalındaki gerçek denetim kayıtlarından görüntülenir.</p>
        </div>
      </div>

      <div className="portal-header flex justify-between items-center">
        <div>
          <h2 className="portal-title">Sistem Yöneticisi Konsolu</h2>
          <p className="portal-subtitle">Platform altyapısı, canari durumu, kullanıcı rolleri ve denetim izleri</p>
        </div>
        <div className="portal-actions">
          <Button
            variant="secondary"
            onClick={fetchHealth}
            loading={loadingHealth}
            aria-label="Sistem Sağlık Kontrolünü Yenile"
          >
            Sağlık Kontrolünü Yenile
          </Button>
        </div>
      </div>

      {/* KPI & Canary Diagnostic Cards */}
      <div className="admin-health-grid grid min-w-0 grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-3 sm:gap-4">
        <div className="admin-health-card kpi-card min-w-0 p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted block">API Sunucu Durumu</span>
          <span className={`break-words font-bold text-lg sm:text-xl block mt-1 ${healthStatus?.status === 'UP' ? 'text-success' : healthStatus?.status === 'DOWN' ? 'text-danger' : 'text-muted'}`}>
            {healthStatus?.status === 'UP' ? 'Aktif (200 OK)' : healthStatus?.status === 'DOWN' ? 'Erişilemiyor' : 'Doğrulanmadı'}
          </span>
          <span className="break-words text-xs text-muted block mt-2">
            Gecikme: {healthLatency !== null ? `${healthLatency} ms` : 'ölçülmedi'} {lastHealthCheck ? `· Kontrol: ${formatDateTime(lastHealthCheck)}` : ''}
          </span>
        </div>

        <div className="admin-health-card kpi-card min-w-0 p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted block">Veritabanı Bağlantısı</span>
          <span className={`break-words font-bold text-lg sm:text-xl block mt-1 ${healthStatus?.database === 'healthy' ? 'text-success' : healthStatus?.status === 'DOWN' ? 'text-danger' : 'text-muted'}`}>
            {healthStatus?.database || 'Doğrulanmadı'}
          </span>
          <span className="text-xs text-muted block mt-2">PostgreSQL 16 · Şema öneki jw_</span>
        </div>

        <div className="admin-health-card kpi-card min-w-0 p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted block">Finansal Defter Güvenliği</span>
          <span className="break-words font-bold text-lg sm:text-xl text-muted block mt-1">Bu kontrolde ölçülmüyor</span>
          <span className="text-xs text-muted block mt-2">Health endpoint ledger mutabakatı yapmaz.</span>
        </div>

        <div className="admin-health-card kpi-card min-w-0 p-4 rounded-lg bg-surface border border-border">
          <span className="text-xs text-muted block">Canary Durumu</span>
          <span className="break-words font-bold text-lg sm:text-xl text-muted block mt-1">Bu kontrolde doğrulanmadı</span>
          <span className="text-xs text-muted block mt-2">Sağlık yanıtı: {healthStatus?.timestamp ? formatDateTime(healthStatus.timestamp) : 'kayıt zamanı bulunamadı'}</span>
        </div>
      </div>

      {/* User and Role Directory */}
      <Card
        title="Sistem Kullanıcıları ve Yetki Matrisi"
        subtitle="Kullanıcılar ve birden çok rol, yetkili API yanıtından alınır"
        action={<Button variant="outline" size="sm" onClick={fetchUsers} loading={loadingUsers}>Yenile</Button>}
      >
        {usersError ? <p role="alert" className="text-sm text-danger">{usersError}</p> : <Table
          ariaLabel="Sistem Kullanıcıları Listesi"
          columns={[
            {
              key: 'email',
              header: 'E-Posta Adresi',
              render: (row) => <span className="font-mono text-sm font-semibold">{row.email}</span>,
            },
            {
              key: 'roles',
              header: 'Sistem Rolü',
              render: (row) => <span className="badge badge-primary">{(row.roles || []).join(', ') || 'Rol yok'}</span>,
            },
            {
              key: 'created',
              header: 'Kayıt Tarihi',
              render: (row) => <span className="text-sm text-muted">{formatDate(row.created_at)}</span>,
            },
          ]}
          data={systemUsers}
          loading={loadingUsers}
        />}
      </Card>

      {/* Audit & Security Log Table */}
      <Card
        title="Güvenlik ve Denetim Günlüğü"
        subtitle="Kalıcı güvenlik anomalileri gerçek tespit zamanlarıyla gösterilir"
        action={
          <Button
            variant="outline"
            size="sm"
            onClick={fetchAnomalies}
            loading={loadingAnomalies}
          >
            Günlüğü Yenile
          </Button>
        }
      >
        {anomaliesError ? <p role="alert" className="text-sm text-danger">{anomaliesError}</p> : anomalies.length > 0 ? (
          <Table
            ariaLabel="Güvenlik Denetim Günlüğü"
            columns={[
              { key: 'id', header: 'Kayıt ID', render: (row) => <span className="font-mono text-xs">{row.id}</span> },
              { key: 'detected_at', header: 'Tarih / Saat', render: (row) => row.detected_at || row.created_at || row.timestamp ? formatDateTime(row.detected_at || row.created_at || row.timestamp) : 'Kayıt zamanı bulunamadı' },
              { key: 'anomaly_type', header: 'Olay Türü', render: (row) => <span className="badge badge-warning">{row.anomaly_type || 'Olay türü bilinmiyor'}</span> },
              { key: 'description', header: 'Açıklama', render: (row) => <span className="text-sm">{row.description || row.message}</span> },
            ]}
            data={anomalies}
          />
        ) : (
          <div className="text-center py-6 text-muted" role={loadingAnomalies ? 'status' : undefined}>
            <p className="text-sm">{loadingAnomalies ? 'Güvenlik günlüğü yükleniyor…' : 'Kayıtlı güvenlik anomalisi bulunmuyor.'}</p>
          </div>
        )}
      </Card>
    </div>
  );
}
