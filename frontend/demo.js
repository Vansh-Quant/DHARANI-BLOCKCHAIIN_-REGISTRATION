(() => {
  window.renderLogin = function renderLogin() {
    const app = document.getElementById('app');
    app.innerHTML = '<div class="login"><div class="login-card"><div class="brand"><div class="brand-mark">◒</div>DHARANI</div><h1>Sign in to DHARANI</h1><p>Choose a demo portal and use OTP 1234.</p><div class="field"><label>Portal</label><select id="demoRole" onchange="applyDemoRole()"><option value="citizen">Citizen Portal</option><option value="authority">Authority Portal</option></select></div><div class="field"><label>Phone number</label><input id="phone" value="9999999999"/></div><div class="field"><label>Email</label><input id="email" value="demo@dharani.local"/></div><div class="actions"><button class="btn btn-primary" onclick="sendOtp()">Send demo OTP</button><button class="btn btn-secondary" onclick="renderLanding()">Back</button></div><div id="otpBox" class="hidden" style="margin-top:18px"><div class="field"><label>OTP</label><div class="otp"><input id="otp" value="1234" maxlength="4"/><button class="btn btn-primary" onclick="verifyOtp()">Verify</button></div></div></div><div class="notice">Authority demo: choose Authority Portal. The backend role controls access.</div></div></div>';
  };
  window.applyDemoRole = function applyDemoRole() {
    const role = document.getElementById('demoRole')?.value;
    const email = document.getElementById('email');
    if (email) email.value = role === 'authority' ? 'authority@dharani.local' : 'demo@dharani.local';
  };
  window.sendOtp = async function sendOtp() {
    try {
      const phone = document.getElementById('phone').value.trim();
      const email = document.getElementById('email').value.trim();
      const data = await request('/auth/register', {method:'POST', body:JSON.stringify({phone_number:phone, first_name:email.startsWith('authority@')?'Demo Authority':'Demo Citizen', last_name:'', email})});
      document.getElementById('otpBox').classList.remove('hidden');
      toast('Demo OTP: ' + (data.demo_otp || '1234'));
    } catch(e) { toast(e.message); }
  };
  window.verifyOtp = async function verifyOtp() {
    try {
      const phone = document.getElementById('phone').value.trim();
      const otp = document.getElementById('otp').value.trim();
      const data = await request('/auth/verify-otp', {method:'POST', body:JSON.stringify({phone_number:phone, otp})});
      saveSession(data);
      await renderDashboard();
    } catch(e) { toast(e.message); }
  };
})();
