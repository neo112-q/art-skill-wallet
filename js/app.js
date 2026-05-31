const API_BASE = '/api/v1';

function getToken() {
  return localStorage.getItem('token') || '';
}

async function apiFetch(path, opts = {}) {
  const headers = { 'Content-Type': 'application/json' };
  const token = getToken();
  if (token) headers['Authorization'] = 'Bearer ' + token;
  const res = await fetch(API_BASE + path, { headers, ...opts });
  const data = await res.json();
  return { ok: res.ok, status: res.status, data };
}
