// Client-side filtering for the tool explorer. Every tool row carries a
// data-search haystack (name + description, lowercased at generation time) and
// a data-access value, so filtering never touches the DOM text.
(function () {
  var search = document.getElementById('tool-search');
  var count = document.getElementById('tool-count');
  var chips = Array.prototype.slice.call(document.querySelectorAll('[data-access-filter]'));
  var tools = Array.prototype.slice.call(document.querySelectorAll('[data-search]'));
  var toolsets = Array.prototype.slice.call(document.querySelectorAll('[data-toolset]'));
  var products = Array.prototype.slice.call(document.querySelectorAll('[data-product]'));
  var explorer = document.getElementById('explorer');
  var total = tools.length;
  var access = 'all';

  function apply() {
    var query = (search.value || '').trim().toLowerCase();
    var terms = query ? query.split(/\s+/) : [];
    var visible = 0;

    tools.forEach(function (tool) {
      var haystack = tool.getAttribute('data-search');
      var ok = (access === 'all' || tool.getAttribute('data-access') === access) &&
        terms.every(function (t) { return haystack.indexOf(t) !== -1; });
      tool.hidden = !ok;
      if (ok) { visible++; }
    });

    // A matrix summarises a whole toolset, so it is only meaningful unfiltered.
    var filtering = terms.length > 0 || access !== 'all';
    toolsets.forEach(function (toolset) {
      var matrix = toolset.querySelector('[data-matrix]');
      if (matrix) { matrix.hidden = filtering; }
      toolset.hidden = !toolset.querySelector('[data-search]:not([hidden])');
    });
    products.forEach(function (product) {
      product.hidden = !product.querySelector('[data-toolset]:not([hidden])');
    });

    explorer.classList.toggle('is-empty', visible === 0);
    count.textContent = visible === total
      ? total + ' tools'
      : visible + ' of ' + total + ' tools';
  }

  search.addEventListener('input', apply);
  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      access = chip.getAttribute('data-access-filter');
      chips.forEach(function (other) {
        other.setAttribute('aria-pressed', String(other === chip));
      });
      apply();
    });
  });

  apply();
})();
