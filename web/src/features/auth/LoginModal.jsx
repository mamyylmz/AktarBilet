import React, { useState } from 'react';
import { Modal } from '../../components/ui/Modal.jsx';
import { Input } from '../../components/ui/Input.jsx';
import { Button } from '../../components/ui/Button.jsx';
import { api } from '../../services/api.js';

export function LoginModal({
  isOpen,
  onClose,
  onLoginSuccess,
  triggerRef,
}) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    if (e) e.preventDefault();
    setError('');

    if (!email || !email.includes('@')) {
      setError('Lütfen geçerli bir e-posta adresi giriniz.');
      return;
    }
    if (!password) {
      setError('Lütfen parolanızı giriniz.');
      return;
    }

    setLoading(true);
    try {
      const data = await api.login(email, password);
      setLoading(false);
      if (data && data.access_token) {
        onLoginSuccess(data.access_token, data.user);
        onClose();
      } else {
        setError('Oturum açma yanıtı geçersiz.');
      }
    } catch (err) {
      setLoading(false);
      setError(err.message || 'E-posta veya parola hatalı.');
    }
  };

  const handleDemoSelect = (demoEmail, demoPassword) => {
    setEmail(demoEmail);
    setPassword(demoPassword);
    setError('');
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="AktarBilet Giriş"
      ariaLabel="AktarBilet Giriş"
      triggerRef={triggerRef}
      maxWidth="480px"
    >
      <form onSubmit={handleSubmit} noValidate>
        {error && (
          <div role="alert" className="alert alert-danger mb-4">
            {error}
          </div>
        )}

        <Input
          id="login-email"
          label="E-Posta Adresi"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          placeholder="ornek@aktarbilet.gov.tr"
          required
          autoComplete="email"
          autoFocus
        />

        <Input
          id="login-password"
          label="Parola"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="••••••••"
          required
          autoComplete="current-password"
        />

        <div className="mt-4">
          <Button
            type="submit"
            variant="primary"
            loading={loading}
            className="w-full"
          >
            Giriş Yap
          </Button>
        </div>

        <div className="demo-accounts-section mt-6 pt-4 border-t border-border">
          <p className="text-xs text-muted mb-2 font-medium">Hızlı Kurumsal Demo Rolleri:</p>
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              className="demo-role-btn"
              onClick={() => handleDemoSelect('yolcu@aktarbilet.gov.tr', 'YolcuPassword123!')}
            >
              Yolcu (Mehmet Demir)
            </button>
            <button
              type="button"
              className="demo-role-btn"
              onClick={() => handleDemoSelect('kontrolor@aktarbilet.gov.tr', 'KontrolorPassword123!')}
            >
              Kontrolör (Ali Saha)
            </button>
            <button
              type="button"
              className="demo-role-btn"
              onClick={() => handleDemoSelect('hatyoneticisi@aktarbilet.gov.tr', 'HatYoneticisi123!')}
            >
              Hat Yöneticisi (Zeynep Hat)
            </button>
            <button
              type="button"
              className="demo-role-btn"
              onClick={() => handleDemoSelect('muhasebe@aktarbilet.gov.tr', 'MuhasebePassword123!')}
            >
              Muhasebe (Canan Finans)
            </button>
            <button
              type="button"
              className="demo-role-btn"
              onClick={() => handleDemoSelect('operasyon@aktarbilet.gov.tr', 'OperasyonPassword123!')}
            >
              Operasyon (Burak Sistem)
            </button>
            <button
              type="button"
              className="demo-role-btn"
              onClick={() => handleDemoSelect('seed.sedir@ddo-case.invalid', 'SedirPassword2026!')}
            >
              Sistem Yöneticisi (Admin)
            </button>
          </div>
        </div>
      </form>
    </Modal>
  );
}
