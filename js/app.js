// app.js — Core API client shared across all pages

const API_BASE = '/api/v1';

// ── Token Storage ─────────────────────────────────────────────────────────────

const TokenStore = {
  getAccess()          { return localStorage.getItem('access_token'); },
  getRefresh()         { return localStorage.getItem('refresh_token'); },
  getRole()            { return localStorage.getItem('role'); },
  getUsername()        { return localStorage.getItem('username'); },

  set(accessToken, refreshToken, role, username) {
    localStorage.setItem('access_token',  accessToken);
    localStorage.setItem('refresh_token', refreshToken);
    localStorage.setItem('role',          role);
    localStorage.setItem('username',      username);
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

  isLoggedIn() {
    return !!this.getAccess();
  },

  isAdmin() {
    return this.getRole() === 'admin';
  },
};

// ── JWT Decode (client-side only — for role/exp reading) ──────────────────────
// NOTE: This does NOT verify the signature. Verification happens server-side.
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
      // Refresh token expired or revoked — force logout
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

// ── apiFetch ──────────────────────────────────────────────────────────────────
// Attaches Bearer token automatically.
// If the access token is expired, silently refreshes it and retries once.

async function apiFetch(path, opts = {}) {
  let accessToken = TokenStore.getAccess();

  // Proactively refresh if the access token is expired
  if (accessToken && tokenIsExpired(accessToken)) {
    const ok = await refreshAccessToken();
    if (!ok) return { ok: false, status: 401, data: { error_message: 'Session expired' } };
    accessToken = TokenStore.getAccess();
  }

  const headers = { 'Content-Type': 'application/json', ...(opts.headers || {}) };
  if (accessToken) {
    headers['Authorization'] = 'Bearer ' + accessToken;
  }

  let res = await fetch(API_BASE + path, { ...opts, headers });

  // If 401, try one silent refresh then retry
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

// ── Auth Actions ──────────────────────────────────────────────────────────────

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
