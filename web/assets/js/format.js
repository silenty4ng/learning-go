/* ==========================================================================
   iGoBlog · format.js — 日期格式化与 HTML 转义
   ========================================================================== */

/** 将 ISO 时间格式化为 YYYY-MM-DD。 */
export function formatDate(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return String(iso);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** 将 ISO 时间格式化为 YYYY-MM-DD HH:mm。 */
export function formatDateTime(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return String(iso);
  const pad = (n) => String(n).padStart(2, '0');
  return `${formatDate(iso)} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** HTML 转义，防止 XSS。 */
export function escapeHtml(str) {
  return String(str ?? '').replace(/[&<>"']/g, (ch) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  }[ch]));
}

/** 数字格式化：超过一万显示 x.x 万。 */
export function formatCount(n) {
  n = Number(n) || 0;
  if (n >= 10000) return `${(n / 10000).toFixed(1)} 万`;
  return String(n);
}
