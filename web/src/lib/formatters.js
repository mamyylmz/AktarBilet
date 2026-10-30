// AktarBilet Frontend Formatters & Integer Currency Utilities

/**
 * Format integer kuruş into Turkish Lira display string
 * @param {number|bigint} kurus Integer amount in kuruş (e.g. 50000 -> "500.00 ₺")
 * @returns {string} Formatted currency string
 */
export function formatKurus(kurus) {
  if (kurus === null || kurus === undefined || isNaN(kurus)) {
    return '0.00 ₺';
  }
  const kurusNum = typeof kurus === 'bigint' ? Number(kurus) : Number(kurus);
  const isNegative = kurusNum < 0;
  const absKurus = Math.abs(kurusNum);
  const tl = Math.floor(absKurus / 100);
  const k = absKurus % 100;
  const formattedTL = tl.toLocaleString('tr-TR');
  const formattedKurus = k.toString().padStart(2, '0');
  return `${isNegative ? '-' : ''}${formattedTL}.${formattedKurus} ₺`;
}

/**
 * Format date string into readable Turkish date and time
 * @param {string|Date} dateStr ISO date string or Date object
 * @returns {string} Formatted date string
 */
export function formatDateTime(dateStr) {
  if (!dateStr) return '—';
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '—';
    return d.toLocaleString('tr-TR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      timeZone: 'Europe/Istanbul',
    });
  } catch {
    return '—';
  }
}

/**
 * Format date string into readable Turkish date only (DD.MM.YYYY)
 * @param {string|Date} dateStr ISO date string or Date object
 * @returns {string} Formatted date string
 */
export function formatDate(dateStr) {
  if (!dateStr) return '—';
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '—';
    return d.toLocaleDateString('tr-TR', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    });
  } catch {
    return '—';
  }
}

/**
 * Format date string into time only (HH:mm:ss)
 */
export function formatTime(dateStr) {
  if (!dateStr) return '—';
  try {
    const d = new Date(dateStr);
    if (isNaN(d.getTime())) return '—';
    return d.toLocaleTimeString('tr-TR', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  } catch {
    return '—';
  }
}

/**
 * Mask card ID for passenger privacy (e.g. "CARD_YOLCU_001" -> "•••• 001")
 */
export function maskCardID(cardId) {
  if (!cardId) return '••••';
  const parts = cardId.split('_');
  const last = parts[parts.length - 1];
  return `•••• ${last}`;
}
