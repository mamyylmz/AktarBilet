import React, { useState } from 'react';

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

        {/* Desktop Navigation */}
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
          <button
            type="button"
            onClick={() => handleNavClick(() => onOpenLegal('how-it-works'))}
            className="nav-link"
          >
            Nasıl Çalışır?
          </button>
          <button
            type="button"
            onClick={() => handleNavClick(() => onOpenLegal('about'))}
            className="nav-link"
          >
            Hakkımızda
          </button>
          <button
            type="button"
            onClick={() => handleNavClick(() => onOpenLegal('help'))}
            className="nav-link"
          >
            Yardım
          </button>
        </nav>

        {/* Auth / Profile, Theme & Hamburger Actions */}
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
            <span className="text-xs hidden lg:inline ml-1">{theme === 'dark' ? 'Açık' : 'Koyu'}</span>
          </button>

          {currentUser ? (
            <div className="user-profile-menu flex items-center gap-2">
              {/* Multi-role Selector */}
              {isMultiRole && (
                <div className="multi-role-selector flex items-center gap-1 bg-surface-muted p-1 rounded-md border border-border">
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

              <div className="user-info text-right hidden sm:block">
                <p className="user-name font-semibold text-xs leading-tight">{currentUser.full_name || currentUser.email}</p>
                <span className="user-role-badge capitalize text-[10px]">{currentUser.role}</span>
              </div>

              <button
                type="button"
                onClick={onLogout}
                className="btn btn-outline btn-sm text-xs"
              >
                Çıkış
              </button>
            </div>
          ) : (
            <button
              ref={loginButtonRef}
              type="button"
              onClick={onOpenLogin}
              className="btn btn-primary btn-sm text-xs px-3"
            >
              Giriş Yap
            </button>
          )}

          {/* Mobile Hamburger Toggle Button */}
          <button
            type="button"
            onClick={() => setMobileMenuOpen((prev) => !prev)}
            className="btn btn-outline btn-sm md:hidden p-1.5"
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

      {/* Mobile Navigation Drawer */}
      {mobileMenuOpen && (
        <div
          id="mobile-nav-drawer"
          className="mobile-nav-drawer md:hidden bg-surface border-b border-border shadow-lg p-4 space-y-2 animate-fadeIn"
        >
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
          </div>
          <div className="pt-2 border-t border-border flex justify-around text-xs text-muted">
            <button type="button" onClick={() => handleNavClick(() => onOpenLegal('how-it-works'))} className="hover:text-primary py-1">
              Nasıl Çalışır?
            </button>
            <button type="button" onClick={() => handleNavClick(() => onOpenLegal('about'))} className="hover:text-primary py-1">
              Hakkımızda
            </button>
            <button type="button" onClick={() => handleNavClick(() => onOpenLegal('help'))} className="hover:text-primary py-1">
              Yardım
            </button>
          </div>
        </div>
      )}
    </header>
  );
}
