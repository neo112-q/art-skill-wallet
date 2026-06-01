// auth.js — Page guards for protected pages

// ── Guards ────────────────────────────────────────────────────────────────────

// Call at the top of every protected page (profile, history, upload, skillManage, editProfile)
function requireAuth() {
  if (!TokenStore.isLoggedIn()) {
    window.location.replace('/home.html');
  }
}

// Call at the top of admin-only pages
function requireAdmin() {
  if (!TokenStore.isLoggedIn()) {
    window.location.replace('/home.html');
    return;
  }
  if (!TokenStore.isAdmin()) {
    window.location.replace('/home.html');
  }
}

// ── Navbar ────────────────────────────────────────────────────────────────────

function populateNavUser() {
  const username = TokenStore.getUsername() || '';
  document.querySelectorAll('.avatar-dropdown-name').forEach(el => {
    el.textContent = username;
  });
}

// ── Password toggle ───────────────────────────────────────────────────────────

function togglePw(inputId, btn) {
  const input = document.getElementById(inputId);
  const hidden = input.type === 'password';
  input.type = hidden ? 'text' : 'password';
  btn.innerHTML = hidden
    ? '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>'
    : '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-5 0-9-4-9-7s4-7 9-7a10.05 10.05 0 011.875.175M15 12a3 3 0 11-6 0 3 3 0 016 0zm6.364-4.364l-14.728 14.728"/></svg>';
}

// ── Auto-wire on DOM ready ────────────────────────────────────────────────────

document.addEventListener('DOMContentLoaded', () => {
  populateNavUser();

  // Wire logout buttons
  document.querySelectorAll('[data-action="logout"]').forEach(el => {
    el.addEventListener('click', (e) => { e.preventDefault(); logout(); });
  });

  // Show/hide admin link based on role
  document.querySelectorAll('[data-admin-only]').forEach(el => {
    el.style.display = TokenStore.isAdmin() ? '' : 'none';
  });
});
