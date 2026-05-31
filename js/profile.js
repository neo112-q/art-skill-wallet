function switchProfileTab(tab) {
  document.querySelectorAll('.profile-tab').forEach(function(btn) {
    btn.classList.toggle('active', btn.dataset.tab === tab);
  });
  document.querySelectorAll('.tab-panel').forEach(function(panel) {
    panel.classList.toggle('active', panel.id === 'panel-' + tab);
  });
}

function openArtwork(el) {
  var d = el.dataset;
  document.getElementById('artwork-img').src           = d.img   || '';
  document.getElementById('artwork-img').alt           = d.title || '';
  document.getElementById('artwork-title').textContent = d.title || '';
  document.getElementById('artwork-date').textContent  = d.date  || '';
  document.getElementById('artwork-skill').textContent = d.skill || '';
  document.getElementById('artwork-level').textContent = d.level || '';
  document.getElementById('artwork-log').textContent   = d.log   || '';
  setPrivacy(d.privacy || 'Public');
  document.getElementById('artworkOverlay').classList.add('open');
  document.body.style.overflow = 'hidden';
}

function closeArtwork() {
  document.getElementById('artworkOverlay').classList.remove('open');
  document.body.style.overflow = '';
}

function setPrivacy(val) {
  document.querySelectorAll('.privacy-btn').forEach(function(btn) {
    btn.classList.toggle('active', btn.dataset.privacy === val);
  });
}

document.addEventListener('DOMContentLoaded', function() {
  document.querySelectorAll('.portfolio-item').forEach(function(item) {
    item.addEventListener('click', function() { openArtwork(this); });
  });

  document.getElementById('artworkOverlay').addEventListener('click', function(e) {
    if (e.target === this) closeArtwork();
  });

  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') closeArtwork();
  });
});
