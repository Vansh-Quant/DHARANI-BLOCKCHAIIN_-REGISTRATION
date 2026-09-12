(() => {
  const API = 'http://localhost:8080/api/v1';
  let frame;
  let ready = false;

  const session = {
    get token() { return localStorage.getItem('dharani_token') || ''; },
    set token(v) { v ? localStorage.setItem('dharani_token', v) : localStorage.removeItem('dharani_token'); },
    get user() { try { return JSON.parse(localStorage.getItem('dharani_user') || 'null'); } catch (_) { return null; } },
    set user(v) { v ? localStorage.setItem('dharani_user', JSON.stringify(v)) : localStorage.removeItem('dharani_user'); }
  };

  async function request(path, options = {}) {
    const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) };
    if (session.token) headers.Authorization = `Bearer ${session.token}`;
    const res = await fetch(API + path, { ...options, headers });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) throw new Error(data.error || `API request failed (${res.status})`);
    return data;
  }

  function toast(message) {
    if (frame?.contentWindow?.toast) frame.contentWindow.toast(message);
    else window.alert(message);
  }

  function els(selector) { return [...frame.contentDocument.querySelectorAll(selector)]; }
  function one(selector) { return frame.contentDocument.querySelector(selector); }
  function show(id) { frame.contentWindow.showPage(id); }

  function userName() {
    const u = session.user || {};
    return [u.first_name, u.last_name].filter(Boolean).join(' ') || 'Citizen';
  }

  function updateHeader() {
    const user = one('.user');
    if (!user) return;
    const name = user.querySelector('span');
    if (name) name.textContent = `Welcome, ${userName()}`;
    const avatar = user.querySelector('.avatar');
    if (avatar) avatar.textContent = (userName()[0] || 'C').toUpperCase();
  }

  function installLogin() {
    const form = one('#login form');
    if (!form) return;
    const fields = form.querySelectorAll('input');
    const phone = fields[0];
    const otp = fields[1];
    const button = form.querySelector('button[type="submit"], .login-btn');
    if (!phone || !otp || !button) return;
    phone.placeholder = 'Phone number';
    phone.inputMode = 'numeric';
    otp.placeholder = 'Demo OTP (1234)';
    otp.maxLength = 4;
    otp.type = 'text';
    form.onsubmit = async (event) => {
      event.preventDefault();
      const number = phone.value.trim();
      if (!/^\d{10,15}$/.test(number)) return toast('Enter a valid phone number.');
      try {
        if (!form.dataset.otpSent) {
          const data = await request('/auth/register', { method: 'POST', body: JSON.stringify({ phone_number: number, first_name: 'Dharani', last_name: 'Citizen', email: `${number}@dharani.local` }) });
          form.dataset.otpSent = '1';
          otp.value = data.demo_otp || '1234';
          button.textContent = 'Verify OTP →';
          toast(`OTP generated: ${data.demo_otp || '1234'}`);
          otp.focus();
          return;
        }
        const data = await request('/auth/verify-otp', { method: 'POST', body: JSON.stringify({ phone_number: number, otp: otp.value.trim() }) });
        session.token = data.token;
        session.user = data.user;
        updateHeader();
        toast('Login successful.');
        await syncAll();
        show('dashboard');
      } catch (error) { toast(error.message); }
    };
  }

  async function syncDashboard() {
    if (!session.token) return;
    const data = await request('/property/user/list');
    const properties = data.properties || [];
    const stats = els('#dashboard .stat strong');
    if (stats[0]) stats[0].textContent = properties.length;
    if (stats[1]) stats[1].textContent = properties.filter(p => ['SUBMITTED','UNDER_REVIEW','CONFLICT','INCONCLUSIVE'].includes(p.verification_status)).length;
    if (stats[2]) stats[2].textContent = properties.filter(p => p.verification_status === 'VERIFIED').length;
    const list = one('#dashboard .properties');
    if (!list) return;
    if (!properties.length) { list.innerHTML = '<div style="padding:22px;color:#6f7c76;font-size:11px">No properties registered yet.</div>'; return; }
    list.innerHTML = properties.slice(0, 6).map(p => `
      <div class="property">
        <div class="thumb"></div>
        <div><div class="property-title">${escapeHtml(p.property_name || 'Property')}</div><div class="meta">Property ID: ${escapeHtml(p.property_id || '')}<br>${escapeHtml(p.location || p.address || '')}</div></div>
        <span class="badge ${badgeClass(p.verification_status)}">${escapeHtml(p.verification_status || 'SUBMITTED')}</span>
        <button class="link" data-passport="${escapeAttr(p.property_id || '')}">View →</button>
      </div>`).join('');
    list.querySelectorAll('[data-passport]').forEach(btn => btn.onclick = async () => { await loadPassport(btn.dataset.passport); show('passport'); });
  }

  async function syncProperties() {
    if (!session.token) return;
    const data = await request('/property/user/list');
    const properties = data.properties || [];
    const card = one('#properties .card');
    if (!card) return;
    const head = card.querySelector('.card-head');
    const list = card.querySelector('.properties');
    if (head) head.querySelector('h2').textContent = `${properties.length} Registered Properties`;
    if (!list) return;
    list.innerHTML = properties.length ? properties.map(p => `<div class="property"><div class="thumb"></div><div><div class="property-title">${escapeHtml(p.property_name || 'Property')}</div><div class="meta">${escapeHtml(p.property_id || '')} · ${escapeHtml(p.location || p.address || '')}</div></div><span class="badge ${badgeClass(p.verification_status)}">${escapeHtml(p.verification_status || '')}</span><button class="link" data-passport="${escapeAttr(p.property_id || '')}">Open Passport →</button></div>`).join('') : '<div style="padding:22px;font-size:11px;color:#6f7c76">No properties registered yet.</div>';
    list.querySelectorAll('[data-passport]').forEach(btn => btn.onclick = async () => { await loadPassport(btn.dataset.passport); show('passport'); });
  }

  function installRegister() {
    const section = one('#register');
    if (!section) return;
    const inputs = [...section.querySelectorAll('input')];
    const selects = [...section.querySelectorAll('select')];
    const next = [...section.querySelectorAll('button')].find(b => /Next/i.test(b.textContent));
    if (!next) return;
    next.onclick = async () => {
      if (!session.token) return show('login');
      const survey = inputs[0]?.value.trim() || '';
      const area = Number(inputs[1]?.value || 0);
      const address = inputs[2]?.value.trim() || '';
      const state = selects[0]?.value || 'Rajasthan';
      const district = selects[1]?.value || '';
      const tehsil = selects[2]?.value || '';
      if (!survey || !area || !address) return toast('Enter survey number, positive area and address.');
      const type = section.querySelector('.type.selected')?.textContent.replace(/\s+/g, ' ').trim() || 'Residential';
      try {
        const data = await request('/property/register', { method: 'POST', body: JSON.stringify({ property_name: `${type.split('(')[0].trim()} Property`, property_type: type, location: `${tehsil}, ${district}, ${state}`, address: `${address}, ${tehsil}, ${district}, ${state}`, area }) });
        toast(`Property ${data.property_id} submitted.`);
        await syncAll();
        show('properties');
      } catch (error) { toast(error.message); }
    };
  }

  async function syncVerification() {
    if (!session.token) return;
    try {
      const data = await request('/verification/queue');
      const card = one('#verification .card');
      if (!card) return;
      const queue = data.queue || [];
      card.innerHTML = queue.length ? queue.map(p => `<div class="queue-item"><div class="thumb"></div><div><div class="property-title">${escapeHtml(p.property_name || 'Property')}</div><div class="meta">${escapeHtml(p.property_id || '')} · ${escapeHtml(p.location || p.address || '')}</div></div><span class="badge yellow">${escapeHtml(p.verification_status || 'PENDING')}</span><button class="review" data-review="${escapeAttr(p.property_id || '')}">Review</button></div>`).join('') : '<div style="padding:22px;font-size:11px;color:#6f7c76">Verification queue is empty.</div>';
      card.querySelectorAll('[data-review]').forEach(btn => btn.onclick = async () => { await review(btn.dataset.review); });
    } catch (error) {
      if (!/Authority role required/.test(error.message)) toast(error.message);
    }
  }

  async function review(id) {
    try {
      const data = await request(`/property/${encodeURIComponent(id)}`);
      const status = data.latest_verification_status || data.verification_status;
      if (status !== 'VERIFIED') {
        toast('Run reconciliation and ensure the report is CLEAR before authority approval.');
        return;
      }
      await request(`/verification/${encodeURIComponent(id)}/review`, { method: 'POST', body: JSON.stringify({ decision: 'VERIFIED', notes: 'Reviewed through DHARANI original frontend.' }) });
      toast('Property verified by authority.');
      await syncAll();
    } catch (error) { toast(error.message); }
  }

  async function loadPassport(id) {
    if (!session.token || !id) return;
    try {
      const data = await request(`/property/${encodeURIComponent(id)}/passport`);
      const p = data.passport || {};
      const infos = one('#passport .info-grid');
      if (infos) {
        const values = infos.querySelectorAll('b');
        if (values[0]) values[0].textContent = p.property_id || '';
        if (values[1]) values[1].textContent = userName();
        if (values[2]) values[2].textContent = p.area || '';
        if (values[3]) values[3].textContent = p.verification_status || '';
        if (values[4]) values[4].textContent = p.property_type || '';
      }
      const title = one('#passport .pass-head h2');
      if (title) title.firstChild.textContent = `${p.property_name || 'Property'} `;
      const timeline = one('#passport .timeline');
      if (timeline && Array.isArray(p.audit_history)) {
        timeline.innerHTML = p.audit_history.map(e => `<div class="event"><div class="dot">✓</div><div><b>${escapeHtml(e.event_type || 'Audit event')}</b><p>${escapeHtml(e.actor || '')} · ${escapeHtml(e.created_at || '')}</p></div></div>`).join('') || '<div class="event"><div class="dot">•</div><div><b>No audit events yet</b></div></div>';
      }
    } catch (error) { toast(error.message); }
  }

  function installPassport() {
    const section = one('#passport');
    if (!section) return;
    const oldView = [...section.querySelectorAll('button')].find(b => /Blockchain Proof/i.test(b.textContent));
    if (oldView) oldView.onclick = () => toast('Blockchain proof is available after the verification anchor step.');
  }

  function installTransfer() {
    const section = one('#transfer');
    if (!section) return;
    const next = [...section.querySelectorAll('button')].find(b => /Next/i.test(b.textContent));
    const inputs = [...section.querySelectorAll('input')];
    if (!next) return;
    next.onclick = async () => {
      if (!session.token) return show('login');
      try {
        const data = await request('/property/user/list');
        const verified = (data.properties || []).find(p => p.verification_status === 'VERIFIED');
        if (!verified) return toast('A VERIFIED property is required before transfer.');
        const buyerPhone = inputs.find(i => /contact/i.test(i.previousElementSibling?.textContent || ''))?.value.trim() || '';
        if (!/^\d{10,15}$/.test(buyerPhone)) return toast('Enter the new owner contact number.');
        const transfer = await request('/transfer/initiate', { method: 'POST', body: JSON.stringify({ property_id: verified.property_id, buyer_phone: buyerPhone, reason: 'Sale' }) });
        toast(`Transfer ${transfer.transfer_id || ''} submitted for authority review.`);
      } catch (error) { toast(error.message); }
    };
  }

  async function syncAll() {
    if (!session.token) return;
    updateHeader();
    await Promise.allSettled([syncDashboard(), syncProperties(), syncVerification()]);
  }

  function badgeClass(status) {
    if (status === 'VERIFIED' || status === 'CLEAR') return 'green';
    if (status === 'CONFLICT' || status === 'REJECTED') return 'red';
    if (status === 'UNDER_REVIEW' || status === 'SUBMITTED') return 'yellow';
    return 'blue';
  }
  function escapeHtml(v) { return String(v ?? '').replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c])); }
  function escapeAttr(v) { return escapeHtml(v); }

  function install() {
    frame = document.getElementById('dharani-original-frame');
    if (!frame || !frame.contentDocument) return;
    ready = true;
    installLogin();
    installRegister();
    installPassport();
    installTransfer();
    updateHeader();
    if (session.token) syncAll();
  }

  window.addEventListener('message', event => {
    if (event.data === 'dharani-original-ready') install();
  });
  window.addEventListener('load', () => setTimeout(install, 100));
})();
