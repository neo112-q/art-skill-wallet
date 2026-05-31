// auth.js — Page guards and UI helpers shared across protected pages

// ── Page Guards ───────────────────────────────────────────────────────────────

// requireAuth — call at the top of every protected page.
// Redirects to home.html if the user is not logged in.
function requireAuth() {
  if (!TokenStore.isLoggedIn()) {
    window.location.replace('/home.html');
  }
}

// requireAdmin — call at the top of admin-only pages.
// Redirects to home.html with a 403-style message if not admin.
function requireAdmin() {
  if (!TokenStore.isLoggedIn()) {
    window.location.replace('/home.html');
    return;
  }
  if (!TokenStore.isAdmin()) {
    window.location.replace('/home.html?error=forbidden');
  }
}

// ── Navbar Population ─────────────────────────────────────────────────────────
// Call populateNavUser() on any page that shows the avatar dropdown
// so the username reflects the logged-in user.

function populateNavUser() {
  const username = TokenStore.getUsername() || 'User';

  const nameEls = document.querySelectorAll('.avatar-dropdown-name');
  nameEls.forEach(el => { el.textContent = username; });
}

// ── Logout wiring ─────────────────────────────────────────────────────────────
// Wire every logout button/element once the DOM is ready.

document.addEventListener('DOMContentLoaded', () => {
  populateNavUser();

  document.querySelectorAll('[data-action="logout"]').forEach(el => {
    el.addEventListener('click', (e) => {
      e.preventDefault();
      logout();
    });
  });

  // Show or hide admin link in dropdown based on role
  const adminLinks = document.querySelectorAll('[data-admin-only]');
  adminLinks.forEach(el => {
    el.style.display = TokenStore.isAdmin() ? '' : 'none';
  });
});

// ── Password visibility toggle ────────────────────────────────────────────────

function togglePw(inputId, btn) {
  const input = document.getElementById(inputId);
  const hidden = input.type === 'password';
  input.type = hidden ? 'text' : 'password';
  btn.innerHTML = hidden
    ? '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>'
    : '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-5 0-9-4-9-7s4-7 9-7a10.05 10.05 0 011.875.175M15 12a3 3 0 11-6 0 3 3 0 016 0zm6.364-4.364l-14.728 14.728"/></svg>';
}
