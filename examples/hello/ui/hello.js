// The hello extension's UI module. The web UI imports it once, before it
// starts; everything it needs comes from window.rackd.
const rackd = window.rackd;

// Alpine components work in markup the extension draws. The UI uses Alpine's
// CSP build, so expressions can only read and call component properties.
rackd.registerComponent('helloCounter', () => ({
  count: 0,
  increment() {
    this.count++;
  },
}));

rackd.registerPage('/hello', (el, ctx) => {
  el.innerHTML = `
    <h1 class="hello-title">Hello</h1>
    <p class="hello-text" data-hello="greeting">Loading...</p>
    <p class="hello-text">You are at <code data-hello="path"></code>.</p>
    <div x-data="helloCounter">
      <button type="button" class="hello-button" @click="increment">Clicked <span x-text="count"></span> times</button>
    </div>`;
  el.querySelector('[data-hello="path"]').textContent = ctx.path;

  // Ignore the answer if the user has left the page by then.
  let left = false;
  rackd
    .request('GET', '/api/v1/hello')
    .then((res) => {
      if (!left) el.querySelector('[data-hello="greeting"]').textContent = res.greeting;
    })
    .catch((err) => rackd.toast.error(err.message));
  return () => {
    left = true;
  };
});

rackd.registerSlot('device-detail', (el, ctx) => {
  if (!rackd.can('hello', 'read')) return;
  const p = document.createElement('p');
  p.className = 'hello-text hello-slot';
  p.textContent = `Hello from ${ctx.data.name}`;
  el.append(p);
});
