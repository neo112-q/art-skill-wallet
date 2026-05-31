document.addEventListener('DOMContentLoaded', function() {
  ['f-skill', 'f-level', 'f-tag'].forEach(function(id) {
    document.getElementById(id).addEventListener('input', filterTimeline);
  });
});

function filterTimeline() {
  var skill = document.getElementById('f-skill').value.toLowerCase();
  var level = document.getElementById('f-level').value.toLowerCase();
  var tag   = document.getElementById('f-tag').value.toLowerCase();

  document.querySelectorAll('.timeline-group').forEach(function(group) {
    var gs = (group.dataset.skill || '').toLowerCase();
    var gl = (group.dataset.level || '').toLowerCase();
    var gt = (group.dataset.tags  || '').toLowerCase();
    var show = (!skill || gs.includes(skill)) &&
               (!level || gl.includes(level)) &&
               (!tag   || gt.includes(tag));
    group.style.display = show ? '' : 'none';
  });
}
