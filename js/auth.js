document.addEventListener('DOMContentLoaded', function() {
  var artPanel = document.getElementById('art-panel');
  if (artPanel) {
    function updateLayout() {
      artPanel.style.display = window.innerWidth >= 768 ? 'block' : 'none';
    }
    updateLayout();
    window.addEventListener('resize', updateLayout);
  }
});

function togglePw(id, btn) {
  var inp = document.getElementById(id);
  var isHidden = inp.type === 'password';
  inp.type = isHidden ? 'text' : 'password';
  btn.innerHTML = isHidden
    ? '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>'
    : '<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-5 0-9-4-9-7s4-7 9-7a10.05 10.05 0 011.875.175M15 12a3 3 0 11-6 0 3 3 0 016 0zm6.364-4.364l-14.728 14.728"/></svg>';
}

function handleLogin(e) {
  e.preventDefault();
  window.location.href = 'profile.html';
}

function handleRegister(e) {
  e.preventDefault();
  var pw  = document.getElementById('reg-password').value;
  var cfm = document.getElementById('reg-confirm') ? document.getElementById('reg-confirm').value : pw;
  var err = document.getElementById('pw-error');
  if (pw !== cfm) {
    if (err) err.style.display = 'block';
    return;
  }
  if (err) err.style.display = 'none';
  window.location.href = 'login.html';
}
