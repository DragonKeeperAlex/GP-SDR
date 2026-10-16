// Client presentation only; this never grants access or changes receiver state.
(() => {
  const requested = new URLSearchParams(location.search).get('display');
  let saved;
  try { saved = localStorage.getItem('gpsdr-ui-layout'); } catch (_) {}
  const valid = value => value === 'touch' || value === 'browser';
  const localPanel = ['127.0.0.1', 'localhost', '[::1]'].includes(location.hostname);
  document.documentElement.dataset.uiLayout = valid(requested) ? requested : valid(saved) ? saved : localPanel ? 'touch' : 'browser';
})();
