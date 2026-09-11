(() => {
  window.renderLogin = function renderLogin() {
    const app = document.getElementById('app');
    app.innerHTML = '<div class="login"><div class="login-card"><div class="brand"><div class="brand-mark">◒</div>DHARANI</div><h1>Sign in to DHARANI</h1><p>Use the demo OTP flow backed by the Go API. No fake local dashboard data is used after login.</p><div class="field"><label>Phone number</label><input id="phone" value="9999999999" placeholder="Enter phone number"/></div><div class="actions"><button class="btn btn-primary" onclick="sendOtp()">Send demo OTP</button><button class="btn btn-secondary" onclick="renderLanding()">Back</button></div><div id="otpBox" class="hidden" style="margin-top:18px"><div class="field"><label>OTP</label><div class="otp"><input id="otp" value="1234" maxlength="4"/><button class="btn btn-primary" onclick="verifyOtp()">Verify</button></div></div></div></div></div>';
  };
  window.applyDemoRole = function() {};
  window.sendOtp = async function sendOtp() {
    try { const phone=document.getElementById('phone').value.trim(); const data=await request('/auth/register',{method:'POST',body:JSON.stringify({phone_number:phone,first_name:'Demo',last_name:'Citizen',email:'demo@dharani.local'})}); document.getElementById('otpBox').classList.remove('hidden'); toast('Demo OTP: '+(data.demo_otp||'1234')); } catch(e){toast(e.message)}
  };
  window.verifyOtp = async function verifyOtp() {
    try { const phone=document.getElementById('phone').value.trim(); const otp=document.getElementById('otp').value.trim(); const data=await request('/auth/verify-otp',{method:'POST',body:JSON.stringify({phone_number:phone,otp})}); saveSession(data); await renderDashboard(); } catch(e){toast(e.message)}
  };
  const s=document.createElement('script'); s.src='presentation.js'; s.onload=()=>window.renderLogin(); document.body.appendChild(s);
})();
