import React, { useState } from 'react';

const ROLE_LABELS = {
  admin: 'Admin',
  hat_yoneticisi: 'Hat Yöneticisi',
  kontrolor: 'Kontrolör',
  muhasebe: 'Muhasebe',
  operasyon: 'Operasyon',
  yolcu: 'Yolcu',
};

export function Navbar({
  currentUser,
  onOpenLogin,
  onLogout,
  onOpenLegal,
  onNavigateHome,
  onOpenMap,
  loginButtonRef,
  activeRole,
  onRoleSelect,
  theme = 'light',
  onToggleTheme,
  onSwitchUserRole,
}) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const isMultiRole = currentUser && Array.isArray(currentUser.roles) && currentUser.roles.length > 1;

  const handleNavClick = (callback) => {
    setMobileMenuOpen(false);
    if (callback) callback();
  };

  return (
    <header className="navbar">
      <div className="navbar-container">
        {/* Sol Alan: Logo & Ana Gezinti */}
        <div className="navbar-left">
          {/* Brand & Logo */}
          <div className="navbar-brand">
            <button
              type="button"
              onClick={() => handleNavClick(onNavigateHome)}
              className="brand-logo-btn focus-visible:outline-2"
              aria-label="AktarBilet Ana Sayfa"
            >
              <span className="brand-icon" aria-hidden="true">🎫</span>
              <span className="brand-name">AktarBilet</span>
            </button>
            <span className="brand-badge hidden sm:inline-block">Kurumsal Ulaşım Portalı</span>
          </div>

          {/* Masaüstü Navigasyon */}
          <nav className="navbar-nav hidden md:flex" aria-label="Ana Gezinti">
            <button
              type="button"
              onClick={() => handleNavClick(onNavigateHome)}
              className={`nav-link ${activeRole === 'home' ? 'active' : ''}`}
              aria-current={activeRole === 'home' ? 'page' : undefined}
            >
              Ana Sayfa
            </button>
            <button
              type="button"
              onClick={() => handleNavClick(() => onRoleSelect('yolcu'))}
              className={`nav-link ${activeRole === 'yolcu' ? 'active' : ''}`}
              aria-current={activeRole === 'yolcu' ? 'page' : undefined}
            >
              Yolcu Portalı
            </button>
            <button
              type="button"
              onClick={() => handleNavClick(onOpenMap)}
              className={`nav-link ${activeRole === 'map' ? 'active' : ''}`}
              aria-current={activeRole === 'map' ? 'page' : undefined}
            >
              Canlı Harita (ESHOT)
            </button>
            <button
              type="button"
              onClick={() => handleNavClick(() => onRoleSelect('heatmap'))}
              className={`nav-link ${activeRole === 'heatmap' ? 'active' : ''}`}
              aria-current={activeRole === 'heatmap' ? 'page' : undefined}
            >
              Isı Haritası
            </button>

            {/* Yetkili Kullanıcı Portal Linkleri */}
            {currentUser && currentUser.role === 'muhasebe' && (
              <button
                type="button"
                onClick={() => handleNavClick(() => onRoleSelect('muhasebe'))}
                className={`nav-link ${activeRole === 'muhasebe' ? 'active' : ''}`}
                aria-current={activeRole === 'muhasebe' ? 'page' : undefined}
              >
                📊 Muhasebe Portalı
              </button>
            )}
            {currentUser && currentUser.role === 'hat_yoneticisi' && (
              <button
                type="button"
                onClick={() => handleNavClick(() => onRoleSelect('hat_yoneticisi'))}
                className={`nav-link ${activeRole === 'hat_yoneticisi' ? 'active' : ''}`}
                aria-current={activeRole === 'hat_yoneticisi' ? 'page' : undefined}
              >
                🚌 Hat Yönetimi
              </button>
            )}
            {currentUser && currentUser.role === 'kontrolor' && (
              <button
                type="button"
                onClick={() => handleNavClick(() => onRoleSelect('kontrolor'))}
                className={`nav-link ${activeRole === 'kontrolor' ? 'active' : ''}`}
                aria-current={activeRole === 'kontrolor' ? 'page' : undefined}
              >
                👮 Kontrolör Portalı
              </button>
            )}
            {currentUser && currentUser.role === 'operasyon' && (
              <button
                type="button"
                onClick={() => handleNavClick(() => onRoleSelect('operasyon'))}
                className={`nav-link ${activeRole === 'operasyon' ? 'active' : ''}`}
                aria-current={activeRole === 'operasyon' ? 'page' : undefined}
              >
                📡 Operasyon Portalı
              </button>
            )}
            {currentUser && currentUser.role === 'admin' && (
              <>
                <button
                  type="button"
                  onClick={() => handleNavClick(() => onRoleSelect('admin'))}
                  className={`nav-link ${activeRole === 'admin' ? 'active' : ''}`}
                  aria-current={activeRole === 'admin' ? 'page' : undefined}
                >
                  ⚙️ Sistem Yönetimi
                </button>
                <button
                  type="button"
                  onClick={() => handleNavClick(() => onRoleSelect('muhasebe'))}
                  className={`nav-link ${activeRole === 'muhasebe' ? 'active' : ''}`}
                  aria-current={activeRole === 'muhasebe' ? 'page' : undefined}
                  title="Muhasebe ve Mutabakat Denetimi"
                >
                  📊 Muhasebe Denetimi
                </button>
              </>
            )}
          </nav>
        </div>

        {/* Sağ Üst Alan: Tema Değiştirici, Kullanıcı Profili, Rol & Çıkış Kontrolleri */}
        <div className="navbar-actions flex items-center gap-2">
          {/* Tema Değiştirici Buton */}
          <button
            type="button"
            onClick={onToggleTheme}
            className="btn btn-outline btn-sm theme-toggle-btn"
            aria-label={`Temayı değiştir (Şu anki: ${theme === 'dark' ? 'Koyu' : 'Açık'})`}
            title={`Temayı değiştir: ${theme === 'dark' ? 'Açık Temaya Geç' : 'Koyu Temaya Geç'}`}
          >
            <span aria-hidden="true">{theme === 'dark' ? '☀️' : '🌙'}</span>
            <span className="text-xs hidden lg:inline ml-1">{theme === 'dark' ? 'Açık' : 'Koyu'}</span>
          </button>

          {/* Giriş Durumuna Göre Kontroller */}
          {currentUser ? (
            <div className="user-profile-menu flex items-center gap-2">
              {/* Çoklu Rol Seçici */}
              {isMultiRole && (
                <div
                  className="multi-role-selector"
                  role="group"
                  aria-label="Yönetici Rol Değiştirici"
                >
                  {currentUser.roles.map((r) => {
                    const isActive = currentUser.role === r;
                    const label = ROLE_LABELS[r] || r;
                    return (
                      <button
                        key={r}
                        type="button"
                        data-role={r}
                        onClick={() => onSwitchUserRole(r)}
                        className={`role-switch-btn ${isActive ? 'active' : ''}`}
                        aria-pressed={isActive}
                        aria-label={`${label} rolüne geç`}
                        title={`${label} rolüne geç`}
                      >
                        {label}
                      </button>
                    );
                  })}
                </div>
              )}

              {/* Kullanıcı Adı ve Rol Bilgisi */}
              <div className="user-info text-right hidden sm:block">
                <p className="user-name font-semibold text-xs leading-tight text-foreground">
                  {currentUser.full_name || currentUser.email}
                </p>
                <span className="user-role-badge capitalize text-[10px] text-brand font-medium">
                  {currentUser.role}
                </span>
              </div>

              {/* Çıkış Butonu */}
              <button
                type="button"
                onClick={onLogout}
                className="btn btn-outline btn-sm text-xs px-3"
              >
                Çıkış
              </button>
            </div>
          ) : (
            <button
              ref={loginButtonRef}
              type="button"
              onClick={onOpenLogin}
              className="btn btn-primary btn-sm text-xs px-3.5 font-medium"
            >
              Giriş Yap
            </button>
          )}

          {/* Mobil Menü Aç/Kapa Butonu (Masaüstünde CSS ile kesinlikle gizlidir) */}
          <button
            type="button"
            onClick={() => setMobileMenuOpen((prev) => !prev)}
            className="navbar-mobile-toggle btn btn-outline btn-sm p-1.5"
            aria-expanded={mobileMenuOpen}
            aria-label={mobileMenuOpen ? 'Menüyü kapat' : 'Menüyü aç'}
            aria-controls="mobile-nav-drawer"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              {mobileMenuOpen ? (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              ) : (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              )}
            </svg>
          </button>
        </div>
      </div>

      {/* Mobil Navigasyon Çekmecesi (<= 768px) */}
      {mobileMenuOpen && (
        <div
          id="mobile-nav-drawer"
          className="mobile-nav-drawer bg-surface border-b border-border shadow-lg p-4 space-y-3 animate-fadeIn"
        >
          {currentUser && (
            <div className="pb-3 border-b border-border flex items-center justify-between">
              <div>
                <p className="font-semibold text-xs text-foreground">{currentUser.full_name || currentUser.email}</p>
                <span className="user-role-badge capitalize text-[10px] text-brand">{currentUser.role}</span>
              </div>
              <button
                type="button"
                onClick={() => handleNavClick(onLogout)}
                className="btn btn-outline btn-sm text-xs px-2.5 py-1"
              >
                Çıkış Yap
              </button>
            </div>
          )}

          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={() => handleNavClick(onNavigateHome)}
              className={`btn btn-sm text-left justify-start w-full ${activeRole === 'home' ? 'btn-primary' : 'btn-outline'}`}
            >
              🏠 Ana Sayfa
            </button>
            <button
              type="button"
              onClick={() => handleNavClick(() => onRoleSelect('yolcu'))}
              className={`btn btn-sm text-left justify-start w-full ${activeRole === 'yolcu' ? 'btn-primary' : 'btn-outline'}`}
            >
              💳 Yolcu Portalı
            </button>
            <button
              type="button"
              onClick={() => handleNavClick(onOpenMap)}
              className={`btn btn-sm text-left justify-start w-full ${activeRole === 'map' ? 'btn-primary' : 'btn-outline'}`}
            >
              🗺️ Canlı Harita
            </button>
            <button
              type="button"
              onClick={() => handleNavClick(() => onRoleSelect('heatmap'))}
              className={`btn btn-sm text-left justify-start w-full ${activeRole === 'heatmap' ? 'btn-primary' : 'btn-outline'}`}
            >
              🔥 Isı Haritası
            </button>

            {currentUser && currentUser.role === 'muhasebe' && (
              <button
                type="button"
                onClick={() => handleNavClick(() => onRoleSelect('muhasebe'))}
                className={`btn btn-sm text-left justify-start w-full col-span-2 ${activeRole === 'muhasebe' ? 'btn-primary' : 'btn-outline'}`}
              >
                📊 Muhasebe Portalı
              </button>
            )}
            {currentUser && currentUser.role === 'admin' && (
              <>
                <button
                  type="button"
                  onClick={() => handleNavClick(() => onRoleSelect('admin'))}
                  className={`btn btn-sm text-left justify-start w-full ${activeRole === 'admin' ? 'btn-primary' : 'btn-outline'}`}
                >
                  ⚙️ Sistem Yönetimi
                </button>
                <button
                  type="button"
                  onClick={() => handleNavClick(() => onRoleSelect('muhasebe'))}
                  className={`btn btn-sm text-left justify-start w-full ${activeRole === 'muhasebe' ? 'btn-primary' : 'btn-outline'}`}
                >
                  📊 Muhasebe Denetimi
                </button>
              </>
            )}
          </div>
        </div>
      )}
    </header>
  );
}
