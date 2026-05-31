// home.js — Auth modal logic for home.html

function openAuth(tab) {
  document.getElementById('authOverlay').classList.add('open');
  switchAuthTab(tab || 'login');
  document.body.style.overflow = 'hidden';
}

function closeAuth() {
  document.getElementById('authOverlay').classList.remove('open');
  document.body.style.overflow = '';
}

function switchAuthTab(tab) {
  ['login', 'register', 'forgot'].forEach(function(t) {
    const tabEl   = document.getElementById('auth-tab-' + t);
    const panelEl = document.getElementById('auth-panel-' + t);
    if (tabEl)   { tabEl.classList.toggle('active', t === tab); tabEl.setAttribute('aria-selected', String(t === tab)); }
    if (panelEl) { panelEl.classList.toggle('active', t === tab); }
  });
  clearAuthMessages();
}

function showForgotPassword() { switchAuthTab('forgot'); }

function showAuthMsg(id, msg, type) {
  const el = document.getElementById(id);
  el.textContent = msg;
  el.className = 'auth-msg ' + (type || 'error');
  el.style.display = 'block';
}

function clearAuthMessages() {
  document.querySelectorAll('.auth-msg').forEach(el => {
    el.style.display = 'none';
    el.textContent   = '';
  });
}

// ── Login ─────────────────────────────────────────────────────────────────────

async function handleLogin() {
  clearAuthMessages();
  const username = document.getElementById('login-username').value.trim();
  const password = document.getElementById('login-password').value;

  if (!username || !password) {
    showAuthMsg('login-msg', 'Please fill in all fields.', 'error');
    return;
  }

  try {
    const result = await apiFetch('/auth/login', {
      method: 'POST',
      body:   JSON.stringify({ username, password }),
    });

    if (result.ok) {
      const { access_token, refresh_token } = result.data.data;

      // Decode role from access token — no sensitive data is stored
      const claims = decodeJWT(access_token);
      const role   = claims?.role || 'user';

      // Store tokens and identity
      TokenStore.set(access_token, refresh_token, role, username);

      closeAuth();

      // Redirect based on role
      window.location.href = role === 'admin' ? '/admin.html' : '/profile.html';

    } else {
      showAuthMsg('login-msg', result.data.error_message || 'Login failed.', 'error');
    }
  } catch {
    showAuthMsg('login-msg', 'Connection error. Please try again.', 'error');
  }
}

// ── Register ──────────────────────────────────────────────────────────────────

async function handleRegister() {
  clearAuthMessages();
  const username = document.getElementById('reg-username').value.trim();
  const password = document.getElementById('reg-password').value;
  const confirm  = document.getElementById('reg-confirm').value;

  if (!username || !password) {
    showAuthMsg('register-msg', 'Username and password are required.', 'error');
    return;
  }
  if (password !== confirm) {
    showAuthMsg('register-msg', 'Passwords do not match.', 'error');
    return;
  }
  if (password.length < 6) {
    showAuthMsg('register-msg', 'Password must be at least 6 characters.', 'error');
    return;
  }

  try {
    const result = await apiFetch('/auth/register', {
      method: 'POST',
      body:   JSON.stringify({ username, hash_pass: password }),
    });

    if (result.ok) {
      switchAuthTab('login');
      document.getElementById('login-username').value = username;
      showAuthMsg('login-msg', 'Account created! Please log in.', 'success');
    } else {
      showAuthMsg('register-msg', result.data.error_message || 'Registration failed.', 'error');
    }
  } catch {
    showAuthMsg('register-msg', 'Connection error. Please try again.', 'error');
  }
}

// ── Forgot Password ───────────────────────────────────────────────────────────

async function handleForgotPassword() {
  clearAuthMessages();
  const email = document.getElementById('forgot-email').value.trim();
  if (!email) {
    showAuthMsg('forgot-msg', 'Please enter your email.', 'error');
    return;
  }
  // Placeholder — wire to real reset endpoint when available
  showAuthMsg('forgot-msg', 'Reset link sent! Check your inbox.', 'success');
}

// ── Google Auth ───────────────────────────────────────────────────────────────

function handleGoogleAuth() {
  // Redirect to backend Google OAuth endpoint when implemented
  window.location.href = '/api/v1/auth/google';
}

// ── Init ──────────────────────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', function() {
  // If already logged in, redirect away from home
  if (TokenStore.isLoggedIn()) {
    const role = TokenStore.getRole();
    window.location.href = role === 'admin' ? '/admin.html' : '/profile.html';
    return;
  }

  // Close modal on backdrop click
  document.getElementById('authOverlay').addEventListener('click', function(e) {
    if (e.target === this) closeAuth();
  });

  // Close modal on Escape key
  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') closeAuth();
  });

  // Smooth scroll for anchor links
  document.querySelectorAll('a[href^="#"]').forEach(a => {
    a.addEventListener('click', function(e) {
      const target = document.querySelector(this.getAttribute('href'));
      if (target) { e.preventDefault(); target.scrollIntoView({ behavior: 'smooth' }); }
    });
  });
});
