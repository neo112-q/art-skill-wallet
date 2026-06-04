// app.js — Core API client shared across all pages

const API_BASE = '/api/v1';

// ── Token Storage ─────────────────────────────────────────────────────────────

const TokenStore = {
  getAccess()   { return localStorage.getItem('access_token'); },
  getRefresh()  { return localStorage.getItem('refresh_token'); },
  getRole()     { return localStorage.getItem('role'); },
  getUsername() { return localStorage.getItem('username'); },

  set(accessToken, refreshToken, role, username) {
    localStorage.setItem('access_token',  accessToken);
    localStorage.setItem('refresh_token', refreshToken);
    localStorage.setItem('role',          role     || 'user');
    localStorage.setItem('username',      username || '');
  },

  setAccess(accessToken) {
    localStorage.setItem('access_token', accessToken);
  },

  clear() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('role');
    localStorage.removeItem('username');
  },

  isLoggedIn() { return !!this.getAccess(); },
  isAdmin()    { return this.getRole() === 'admin'; },
};

// ── JWT Decode ────────────────────────────────────────────────────────────────

function decodeJWT(token) {
  try {
    const payload = token.split('.')[1];
    return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')));
  } catch {
    return null;
  }
}

function tokenIsExpired(token) {
  const claims = decodeJWT(token);
  if (!claims || !claims.exp) return true;
  return Date.now() / 1000 >= claims.exp;
}

// ── Token Refresh ─────────────────────────────────────────────────────────────

async function refreshAccessToken() {
  const refreshToken = TokenStore.getRefresh();
  if (!refreshToken) return false;

  try {
    const res = await fetch(API_BASE + '/auth/refresh', {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) {
      TokenStore.clear();
      window.location.href = '/home.html';
      return false;
    }

    const body = await res.json();
    TokenStore.setAccess(body.data.access_token);
    return true;

  } catch {
    return false;
  }
}

// ── Ensure valid token (shared pre-flight) ───────────────────────────────────

async function ensureToken() {
  let accessToken = TokenStore.getAccess();
  if (accessToken && tokenIsExpired(accessToken)) {
    const ok = await refreshAccessToken();
    if (!ok) return null;
    accessToken = TokenStore.getAccess();
  }
  return accessToken;
}

// ── apiFetch — JSON requests ─────────────────────────────────────────────────

async function apiFetch(path, opts = {}) {
  let accessToken = await ensureToken();
  if (!accessToken && TokenStore.getAccess()) {
    return { ok: false, status: 401, data: { error_message: 'Session expired' } };
  }

  const headers = {
    'Content-Type': 'application/json',
    ...(opts.headers || {}),
  };
  if (accessToken) {
    headers['Authorization'] = 'Bearer ' + accessToken;
  }

  let res = await fetch(API_BASE + path, { ...opts, headers });

  // Silent refresh + one retry on 401
  if (res.status === 401) {
    const ok = await refreshAccessToken();
    if (!ok) return { ok: false, status: 401, data: { error_message: 'Session expired' } };
    headers['Authorization'] = 'Bearer ' + TokenStore.getAccess();
    res = await fetch(API_BASE + path, { ...opts, headers });
  }

  let data = {};
  try { data = await res.json(); } catch { /* empty body */ }

  return { ok: res.ok, status: res.status, data };
}

// ── apiUpload — multipart/form-data requests ─────────────────────────────────
// Do NOT set Content-Type manually — the browser sets it with the boundary.

async function apiUpload(path, formData, method = 'POST') {
  let accessToken = await ensureToken();
  if (!accessToken) {
    return { ok: false, status: 401, data: { error_message: 'Session expired' } };
  }

  let res = await fetch(API_BASE + path, {
    method,
    headers: { 'Authorization': 'Bearer ' + accessToken },
    body: formData,
  });

  if (res.status === 401) {
    const ok = await refreshAccessToken();
    if (!ok) return { ok: false, status: 401, data: { error_message: 'Session expired' } };
    res = await fetch(API_BASE + path, {
      method,
      headers: { 'Authorization': 'Bearer ' + TokenStore.getAccess() },
      body: formData,
    });
  }

  let data = {};
  try { data = await res.json(); } catch {}

  return { ok: res.ok, status: res.status, data };
}

// ── Logout ────────────────────────────────────────────────────────────────────

async function logout() {
  const refreshToken = TokenStore.getRefresh();
  if (refreshToken) {
    await apiFetch('/auth/logout', {
      method: 'POST',
      body:   JSON.stringify({ refresh_token: refreshToken }),
    });
  }
  TokenStore.clear();
  window.location.href = '/home.html';
}

// ── Toast notifications (shared across all pages) ─────────────────────────────
// Usage: showToast('Saved!', 'success') | showToast('Failed', 'error') | showToast('msg')

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, m =>
    ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m]));
}

function showToast(message, type = 'info', duration = 3500) {
  let container = document.getElementById('asw-toasts');
  if (!container) {
    container = document.createElement('div');
    container.id = 'asw-toasts';
    container.style.cssText =
      'position:fixed;top:18px;right:18px;z-index:99999;display:flex;flex-direction:column;gap:10px;max-width:360px;pointer-events:none;';
    document.body.appendChild(container);
    const style = document.createElement('style');
    style.textContent =
      '@keyframes aswToastIn{from{opacity:0;transform:translateX(40px)}to{opacity:1;transform:translateX(0)}}' +
      '@keyframes aswToastOut{to{opacity:0;transform:translateX(40px)}}';
    document.head.appendChild(style);
  }

  const themes = {
    success: { bg: '#15301b', border: '#2e7d32', fg: '#86efac', icon: '✓' },
    error:   { bg: '#341515', border: '#c0392b', fg: '#ff9b8a', icon: '✕' },
    info:    { bg: '#1c1c1c', border: '#2A2A2A', fg: '#dddddd', icon: 'ℹ' },
  };
  const t = themes[type] || themes.info;

  const toast = document.createElement('div');
  toast.style.cssText =
    `pointer-events:auto;display:flex;align-items:flex-start;gap:10px;background:${t.bg};` +
    `border:1px solid ${t.border};color:${t.fg};padding:12px 14px;border-radius:10px;` +
    `font-size:0.85rem;font-family:'Syne',sans-serif;line-height:1.45;` +
    `box-shadow:0 8px 28px rgba(0,0,0,0.45);animation:aswToastIn .25s ease;cursor:pointer;`;
  toast.innerHTML =
    `<span style="font-weight:800;flex-shrink:0;">${t.icon}</span>` +
    `<span>${escapeHtml(message)}</span>`;

  const remove = () => {
    toast.style.animation = 'aswToastOut .25s ease forwards';
    setTimeout(() => toast.remove(), 250);
  };
  toast.addEventListener('click', remove); // click to dismiss early
  container.appendChild(toast);
  setTimeout(remove, duration);
}
