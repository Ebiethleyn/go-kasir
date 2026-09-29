const API = '/api';
const THEME_KEY = 'go-kasir-theme';

function applyTheme(theme) {
  const nextTheme = theme === 'dark' ? 'dark' : 'light';
  document.documentElement.dataset.theme = nextTheme;
  try { localStorage.setItem(THEME_KEY, nextTheme); } catch {}
  document.querySelectorAll('[data-theme-toggle]').forEach(button => {
    const darkMode = nextTheme === 'dark';
    button.setAttribute('aria-label', darkMode ? 'Gunakan mode terang' : 'Gunakan mode gelap');
    button.setAttribute('title', darkMode ? 'Gunakan mode terang' : 'Gunakan mode gelap');
    button.innerHTML = `<i class="bi ${darkMode ? 'bi-sun' : 'bi-moon-stars'}"></i><span>${darkMode ? 'Mode terang' : 'Mode gelap'}</span>`;
  });
  document.dispatchEvent(new CustomEvent('themechange', { detail: { theme: nextTheme } }));
}

let preferredTheme = 'light';
try { preferredTheme = localStorage.getItem(THEME_KEY) || 'light'; } catch {}
applyTheme(preferredTheme);

document.addEventListener('click', event => {
  if (!event.target.closest('[data-theme-toggle]')) return;
  applyTheme(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark');
});

const Auth = {
  getToken() { return localStorage.getItem('access_token'); },
  getRefresh() { return localStorage.getItem('refresh_token'); },
  getOwner() {
    try { return JSON.parse(localStorage.getItem('owner') || 'null'); } catch { return null; }
  },
  setSession(data) {
    localStorage.setItem('access_token', data.access_token);
    localStorage.setItem('refresh_token', data.refresh_token);
    localStorage.setItem('owner', JSON.stringify(data.owner));
    localStorage.setItem('expires_at', data.expires_at);
  },
  clear() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('owner');
    localStorage.removeItem('expires_at');
  },
  require() {
    if (!this.getToken()) {
      window.location.href = 'index.html';
      return false;
    }
    return true;
  }
};

async function api(path, options = {}) {
  const opts = { ...options };
  opts.headers = { ...(options.headers || {}) };
  if (!(options.body instanceof FormData)) {
    opts.headers['Content-Type'] = opts.headers['Content-Type'] || 'application/json';
  }
  const token = Auth.getToken();
  if (token) opts.headers['Authorization'] = 'Bearer ' + token;

  let res = await fetch(API + path, opts);
  if (res.status === 401 && Auth.getRefresh() && !options._retry) {
    const refreshed = await refreshToken();
    if (refreshed) {
      return api(path, { ...options, _retry: true });
    }
    Auth.clear();
    window.location.href = 'index.html';
    throw new Error('Sesi berakhir');
  }

  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/json')) {
    const body = await res.json();
    if (!res.ok || body.success === false) {
      throw new Error(body.message || 'Terjadi kesalahan');
    }
    return body;
  }
  if (!res.ok) throw new Error('Request gagal');
  return res;
}

async function refreshToken() {
  try {
    const res = await fetch(API + '/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: Auth.getRefresh() })
    });
    const body = await res.json();
    if (!res.ok || !body.success) return false;
    Auth.setSession(body.data);
    return true;
  } catch {
    return false;
  }
}

async function downloadAuth(path, filename) {
  const res = await fetch(API + path, {
    headers: { Authorization: 'Bearer ' + Auth.getToken() }
  });
  if (res.status === 401) {
    Auth.clear();
    window.location.href = 'index.html';
    return;
  }
  if (!res.ok) {
    let msg = 'Gagal mengunduh';
    try {
      const j = await res.json();
      msg = j.message || msg;
    } catch {}
    throw new Error(msg);
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

function formatRp(n) {
  const v = Number(n) || 0;
  return 'Rp ' + v.toLocaleString('id-ID', { maximumFractionDigits: 0 });
}

function qs(sel) { return document.querySelector(sel); }
function qsa(sel) { return [...document.querySelectorAll(sel)]; }

function toast(msg, type = 'success') {
  let el = qs('#appToast');
  if (!el) {
    el = document.createElement('div');
    el.id = 'appToast';
    el.className = 'toast align-items-center position-fixed bottom-0 end-0 m-3';
    el.setAttribute('role', 'alert');
    el.innerHTML = `<div class="d-flex"><div class="toast-body"></div><button type="button" class="btn-close me-2 m-auto" data-bs-dismiss="toast"></button></div>`;
    document.body.appendChild(el);
  }
  el.classList.remove('text-bg-success', 'text-bg-danger');
  el.classList.add(type === 'error' ? 'text-bg-danger' : 'text-bg-success');
  el.querySelector('.toast-body').textContent = msg;
  bootstrap.Toast.getOrCreateInstance(el, { delay: 3000 }).show();
}

function todayStr() {
  return new Date().toISOString().slice(0, 10);
}

function daysAgoStr(n) {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.toISOString().slice(0, 10);
}

function renderShell(active) {
  const owner = Auth.getOwner() || {};
  const links = [
    { href: 'dashboard.html', icon: 'bi-speedometer2', label: 'Dashboard', key: 'dashboard' },
    { href: 'statistics.html', icon: 'bi-bar-chart-line', label: 'Statistika Toko', key: 'statistics' },
    { href: 'finance.html', icon: 'bi-cash-coin', label: 'Likuiditas & Arus Kas', key: 'finance' },
    { href: 'pos.html', icon: 'bi-cart3', label: 'Kasir (POS)', key: 'pos' },
    { href: 'products.html', icon: 'bi-box-seam', label: 'Produk', key: 'products' },
    { href: 'customers.html', icon: 'bi-people', label: 'Pelanggan', key: 'customers' },
    { href: 'transactions.html', icon: 'bi-receipt', label: 'Transaksi', key: 'transactions' },
    { href: 'reports.html', icon: 'bi-graph-up', label: 'Laporan', key: 'reports' },
    { href: 'backup.html', icon: 'bi-cloud-arrow-down', label: 'Backup', key: 'backup' },
    { href: 'settings.html', icon: 'bi-gear', label: 'Pengaturan', key: 'settings' },
  ];
  const nav = links.map(l =>
    `<a href="${l.href}" class="${l.key === active ? 'active' : ''}"><i class="bi ${l.icon}"></i> ${l.label}</a>`
  ).join('');

  return `
  <div class="sidebar-backdrop" id="sidebarBackdrop"></div>
  <aside class="sidebar" id="sidebar">
    <div class="brand"><span class="brand-mark"><i class="bi bi-bag-check-fill"></i></span><span>Go Kasir<small>Point of Sale</small></span></div>
    <nav class="flex-grow-1">${nav}</nav>
    <div class="p-3 border-top border-secondary">
      <div class="small text-secondary mb-2">${owner.full_name || owner.username || 'Owner'}</div>
      <button class="btn btn-sm btn-outline-light theme-toggle w-100 mb-2" type="button" data-theme-toggle></button>
      <button class="btn btn-sm btn-outline-light w-100" id="btnLogout"><i class="bi bi-box-arrow-right"></i> Logout</button>
    </div>
  </aside>`;
}

function initShell(active, title) {
  if (!Auth.require()) return false;
  const shell = qs('#appShell');
  if (shell) {
    shell.insertAdjacentHTML('afterbegin', renderShell(active));
    applyTheme(document.documentElement.dataset.theme);
  }
  const titleEl = qs('#pageTitle');
  if (titleEl) titleEl.textContent = title;
  const ownerEl = qs('#topOwner');
  if (ownerEl) {
    const o = Auth.getOwner() || {};
    ownerEl.textContent = o.full_name || o.username || '';
  }
  qs('#btnLogout')?.addEventListener('click', () => {
    Auth.clear();
    location.href = 'index.html';
  });
  qs('#btnMenu')?.addEventListener('click', () => {
    qs('#sidebar')?.classList.toggle('open');
    qs('#sidebarBackdrop')?.classList.toggle('show');
  });
  qs('#sidebarBackdrop')?.addEventListener('click', () => {
    qs('#sidebar')?.classList.remove('open');
    qs('#sidebarBackdrop')?.classList.remove('show');
  });
  return true;
}
