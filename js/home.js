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
  ['login', 'register'].forEach(function(t) {
    document.getElementById('auth-tab-' + t).classList.toggle('active', t === tab);
    document.getElementById('auth-tab-' + t).setAttribute('aria-selected', String(t === tab));
    document.getElementById('auth-panel-' + t).classList.toggle('active', t === tab);
  });
  clearAuthMessages();
}

function showAuthMsg(id, msg, type) {
  var el = document.getElementById(id);
  el.textContent = msg;
  el.className = 'auth-msg ' + (type || 'error');
  el.style.display = 'block';
}

function clearAuthMessages() {
  document.querySelectorAll('.auth-msg').forEach(function(el) {
    el.style.display = 'none';
    el.textContent = '';
  });
}

async function handleLogin() {
  clearAuthMessages();
  var username = document.getElementById('login-username').value.trim();
  var password = document.getElementById('login-password').value;
  if (!username || !password) {
    showAuthMsg('login-msg', 'Please fill in all fields.', 'error');
    return;
  }
  try {
    var result = await apiFetch('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username: username, password: password })
    });
    if (result.ok) {
      if (result.data.data && result.data.data.token) {
        localStorage.setItem('token', result.data.data.token);
      }
      closeAuth();
      window.location.href = 'profile.html';
    } else {
      showAuthMsg('login-msg', result.data.error_message || 'Login failed.', 'error');
    }
  } catch (e) {
    showAuthMsg('login-msg', 'Connection error. Please try again.', 'error');
  }
}

async function handleRegister() {
  clearAuthMessages();
  var username = document.getElementById('reg-username').value.trim();
  var bio      = document.getElementById('reg-bio').value.trim();
  var password = document.getElementById('reg-password').value;
  var confirm  = document.getElementById('reg-confirm').value;
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
    var result = await apiFetch('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username: username, bio: bio, hash_pass: password })
    });
    if (result.ok) {
      switchAuthTab('login');
      document.getElementById('login-username').value = username;
      showAuthMsg('login-msg', 'Account created! Please login.', 'success');
    } else {
      showAuthMsg('register-msg', result.data.error_message || 'Registration failed.', 'error');
    }
  } catch (e) {
    showAuthMsg('register-msg', 'Connection error. Please try again.', 'error');
  }
}

document.addEventListener('DOMContentLoaded', function() {
  document.getElementById('authOverlay').addEventListener('click', function(e) {
    if (e.target === this) closeAuth();
  });

  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') closeAuth();
  });

  document.querySelectorAll('a[href^="#"]').forEach(function(a) {
    a.addEventListener('click', function(e) {
      var target = document.querySelector(this.getAttribute('href'));
      if (target) { e.preventDefault(); target.scrollIntoView({ behavior: 'smooth' }); }
    });
  });
});
