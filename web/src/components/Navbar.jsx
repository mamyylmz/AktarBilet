import React from 'react';

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
  const isMultiRole = currentUser && Array.isArray(currentUser.roles) && currentUser.roles.length > 1;

  return (
    <header className="navbar">
      <div className="navbar-container">
        {/* Brand & Logo */}
        <div className="navbar-brand">
          <button
            type="button"
            onClick={onNavigateHome}
            className="brand-logo-btn focus-visible:outline-2"
            aria-label="AktarBilet Ana Sayfa"
          >
            <span className="brand-icon" aria-hidden="true">🎫</span>
            <span className="brand-name">AktarBilet</span>
          </button>
          <span className="brand-badge hidden sm:inline-block">Kurumsal Ulaşım Portalı</span>
        </div>

        {/* Public / Role Navigation */}
        <nav className="navbar-nav" aria-label="Ana Gezinti">
          <button
            type="button"
            onClick={onNavigateHome}
            className={`nav-link ${activeRole === 'home' ? 'active' : ''}`}
            aria-current={activeRole === 'home' ? 'page' : undefined}
          >
            Ana Sayfa
          </button>
          <button
            type="button"
            onClick={() => onRoleSelect('yolcu')}
            className={`nav-link ${activeRole === 'yolcu' ? 'active' : ''}`}
            aria-current={activeRole === 'yolcu' ? 'page' : undefined}
          >
            Yolcu Portalı
          </button>
          <button
            type="button"
            onClick={onOpenMap}
            className={`nav-link ${activeRole === 'map' ? 'active' : ''}`}
            aria-current={activeRole === 'map' ? 'page' : undefined}
          >
            Canlı Harita (ESHOT)
          </button>
          <button
            type="button"
            onClick={() => onRoleSelect('heatmap')}
            className={`nav-link ${activeRole === 'heatmap' ? 'active' : ''}`}
            aria-current={activeRole === 'heatmap' ? 'page' : undefined}
          >
            Isı Haritası
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('how-it-works')}
            className="nav-link"
          >
            Nasıl Çalışır?
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('about')}
            className="nav-link"
          >
            Hakkımızda
          </button>
          <button
            type="button"
            onClick={() => onOpenLegal('help')}
            className="nav-link"
          >
            Yardım
          </button>
        </nav>

        {/* Auth / Profile & Theme Actions */}
        <div className="navbar-actions flex items-center gap-2">
          {/* Theme Toggle Button */}
          <button
            type="button"
            onClick={onToggleTheme}
            className="btn btn-outline btn-sm theme-toggle-btn"
            aria-label={`Temayı değiştir (Şu anki: ${theme === 'dark' ? 'Koyu' : 'Açık'})`}
            title={`Temayı değiştir: ${theme === 'dark' ? 'Açık Temaya Geç' : 'Koyu Temaya Geç'}`}
          >
            <span aria-hidden="true">{theme === 'dark' ? '☀️' : '🌙'}</span>
            <span className="text-xs hidden md:inline ml-1">{theme === 'dark' ? 'Açık' : 'Koyu'}</span>
          </button>

          {currentUser ? (
            <div className="user-profile-menu flex items-center gap-3">
              {/* Multi-role Selector if user has multiple roles */}
              {isMultiRole && (
                <div className="multi-role-selector flex items-center gap-1 bg-surface-muted p-1 rounded-md border border-border">
                  <span className="text-[11px] text-muted font-medium px-1 hidden lg:inline">Rol:</span>
                  {currentUser.roles.map((r) => (
                    <button
                      key={r}
                      type="button"
                      onClick={() => onSwitchUserRole(r)}
                      className={`text-xs px-2 py-0.5 rounded font-medium transition-colors ${
                        currentUser.role === r
                          ? 'bg-brand text-white shadow-sm'
                          : 'text-muted hover:text-foreground'
                      }`}
                      aria-label={`${r} rolüne geç`}
                    >
                      {r === 'admin' ? 'Admin' : r === 'yolcu' ? 'Yolcu' : r}
                    </button>
                  ))}
                </div>
              )}

              <div className="user-info text-right hidden md:block">
                <p className="user-name font-semibold text-sm">{currentUser.full_name || currentUser.email}</p>
                <span className="user-role-badge capitalize">{currentUser.role}</span>
              </div>

              <button
                type="button"
                onClick={onLogout}
                className="btn btn-outline btn-sm"
              >
                Çıkış Yap
              </button>
            </div>
          ) : (
            <button
              ref={loginButtonRef}
              type="button"
              onClick={onOpenLogin}
              className="btn btn-primary btn-sm"
            >
              Giriş Yap
            </button>
          )}
        </div>
      </div>
    </header>
  );
}
