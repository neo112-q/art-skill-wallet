document.addEventListener('DOMContentLoaded', function() {
  ['f-name', 'f-skill', 'f-level', 'f-tag'].forEach(function(id) {
    document.getElementById(id).addEventListener('input', filterArtists);
  });
});

function filterArtists() {
  var name  = document.getElementById('f-name').value.toLowerCase();
  var skill = document.getElementById('f-skill').value.toLowerCase();
  var level = document.getElementById('f-level').value.toLowerCase();
  var tag   = document.getElementById('f-tag').value.toLowerCase();

  document.querySelectorAll('.artist-card').forEach(function(card) {
    var cn = (card.dataset.name   || '').toLowerCase();
    var cs = (card.dataset.skills || '').toLowerCase();
    var cl = (card.dataset.level  || '').toLowerCase();
    var ct = (card.dataset.tags   || '').toLowerCase();
    var show = (!name  || cn.includes(name))  &&
               (!skill || cs.includes(skill)) &&
               (!level || cl.includes(level)) &&
               (!tag   || ct.includes(tag));
    card.style.display = show ? '' : 'none';
  });
}
