// Global Telemedicine Real-Time WebSocket Client

(function() {
  const token = localStorage.getItem('ms_token');
  if (!token) return;

  const user = JSON.parse(localStorage.getItem('ms_user') || '{}');
  const role = user.role || 'patient';
  const userID = user.id;

  let ws = null;
  let ringInterval = null;
  let ringAudioContext = null;

  // CSS injection for premium overlays
  const style = document.createElement('style');
  style.textContent = `
    @keyframes tm-pulse { 0%, 100% { transform: scale(1); box-shadow: 0 0 0 0 rgba(15, 157, 88, 0.6); } 50% { transform: scale(1.06); box-shadow: 0 0 0 16px rgba(15, 157, 88, 0); } }
    @keyframes tm-wave { 0% { transform: scale(1); opacity: 0.7; } 100% { transform: scale(2.2); opacity: 0; } }
    .tm-overlay { position: fixed; inset: 0; z-index: 99999; background: rgba(10, 10, 10, 0.9); backdrop-filter: blur(8px); display: flex; flex-direction: column; align-items: center; justify-content: center; font-family: 'Inter', system-ui, sans-serif; color: #fff; }
    .tm-ring-container { position: relative; width: 120px; height: 120px; display: flex; align-items: center; justify-content: center; margin-bottom: 24px; }
    .tm-wave { position: absolute; inset: 0; border-radius: 50%; border: 3px solid rgba(15, 157, 88, 0.4); animation: tm-wave 1.8s ease-out infinite; }
    .tm-wave:nth-child(2) { animation-delay: 0.6s; }
    .tm-wave:nth-child(3) { animation-delay: 1.2s; }
    .tm-avatar { width: 80px; height: 80px; border-radius: 50%; background: linear-gradient(135deg, #0F9D58, #34A853); display: flex; align-items: center; justify-content: center; font-size: 1.8rem; font-weight: 700; color: #fff; position: relative; z-index: 2; animation: tm-pulse 1.8s ease-in-out infinite; }
    .tm-btn { width: 64px; height: 64px; border-radius: 50%; border: none; cursor: pointer; display: flex; align-items: center; justify-content: center; transition: transform 0.1s, box-shadow 0.2s; }
    .tm-btn:active { transform: scale(0.95); }
    .tm-btn-accept { background: #10b981; box-shadow: 0 4px 20px rgba(16, 185, 129, 0.4); }
    .tm-btn-decline { background: #ef4444; box-shadow: 0 4px 20px rgba(239, 68, 68, 0.4); }
  `;
  document.head.appendChild(style);

  function connect() {
    const protocol = location.protocol === 'https:' ? 'wss://' : 'ws://';
    const wsURL = `${protocol}${location.host}/api/v1/ws?token=${token}`;

    ws = new WebSocket(wsURL);

    ws.onmessage = function(e) {
      try {
        const msg = JSON.parse(e.data);
        handleMessage(msg);
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    ws.onclose = function() {
      // Reconnect after 3 seconds
      setTimeout(connect, 3000);
    };
  }

  function handleMessage(msg) {
    if (role === 'doctor' && msg.type === 'incoming_call') {
      showIncomingCallOverlay(msg.payload);
    } else if (role === 'patient') {
      if (msg.type === 'call_accepted') {
        // Dispatch custom event to notify components on the appointments page
        window.dispatchEvent(new CustomEvent('ms-call-accepted', { detail: msg.payload }));
      } else if (msg.type === 'call_declined') {
        window.dispatchEvent(new CustomEvent('ms-call-declined', { detail: msg.payload }));
      }
    }
  }

  // ─── Doctor Ringing Tone (Web Audio API) ─────────────────────────────────────
  function startRinging() {
    stopRinging();
    const playBeep = () => {
      try {
        if (!ringAudioContext) {
          ringAudioContext = new (window.AudioContext || window.webkitAudioContext)();
        }
        const ctx = ringAudioContext;
        const now = ctx.currentTime;

        const beep = (start, freq) => {
          const osc = ctx.createOscillator();
          const gain = ctx.createGain();
          osc.connect(gain);
          gain.connect(ctx.destination);
          osc.frequency.value = freq;
          osc.type = 'sine';
          gain.gain.setValueAtTime(0.001, start);
          gain.gain.exponentialRampToValueAtTime(0.25, start + 0.05);
          gain.gain.setValueAtTime(0.25, start + 0.35);
          gain.gain.exponentialRampToValueAtTime(0.001, start + 0.4);
          osc.start(start);
          osc.stop(start + 0.4);
        };

        beep(now, 480);
        beep(now + 0.5, 480);
      } catch (e) {
        console.warn('AudioContext failed:', e);
      }
    };

    playBeep();
    ringInterval = setInterval(playBeep, 2500);
  }

  function stopRinging() {
    if (ringInterval) {
      clearInterval(ringInterval);
      ringInterval = null;
    }
    if (ringAudioContext) {
      try { ringAudioContext.close(); } catch(_) {}
      ringAudioContext = null;
    }
  }

  // ─── Incoming Call Overlay for Doctors ───────────────────────────────────────
  function showIncomingCallOverlay(appt) {
    if (document.getElementById('tm-incoming-overlay')) return;

    // Start ringing
    startRinging();

    const patientName = appt.patient ? `${appt.patient.user.first_name} ${appt.patient.user.last_name}` : 'Patient';
    const initials = patientName.split(' ').map(w => w[0] || '').join('').slice(0, 2).toUpperCase();
    const typeLabel = appt.type === 'chat' ? 'Chat Consultation' : appt.type === 'voice' ? 'Voice Call' : 'Video Call';

    const overlay = document.createElement('div');
    overlay.id = 'tm-incoming-overlay';
    overlay.className = 'tm-overlay';
    overlay.innerHTML = `
      <div style="text-align: center; max-width: 320px; width: 100%; padding: 20px;">
        <div class="tm-ring-container">
          <div class="tm-wave"></div>
          <div class="tm-wave"></div>
          <div class="tm-wave"></div>
          <div class="tm-avatar">${initials}</div>
        </div>
        <p style="color: #86EFAC; font-size: 0.8rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; margin-bottom: 8px;">Incoming consultation request</p>
        <h3 style="font-size: 1.4rem; font-weight: 800; margin-bottom: 4px;">${patientName}</h3>
        <p style="color: #9CA3AF; font-size: 0.85rem; margin-bottom: 12px;">Reason: ${appt.chief_complaint || 'No complaint specified'}</p>
        <p style="color: #6B7280; font-size: 0.8rem; margin-bottom: 36px;">Type: ${typeLabel}</p>
        
        <div style="display: flex; gap: 24px; justify-content: center;">
          <!-- Decline -->
          <div style="display: flex; flex-direction: column; align-items: center; gap: 8px;">
            <button id="tm-btn-decline" class="tm-btn tm-btn-decline">
              <svg width="24" height="24" fill="white" viewBox="0 0 24 24" style="transform: rotate(135deg);">
                <path d="M20.01 15.38c-1.23 0-2.42-.2-3.53-.56-.35-.12-.74-.03-1.01.24l-1.57 1.97c-2.83-1.35-5.48-3.9-6.89-6.83l1.95-1.66c.27-.28.35-.67.24-1.02-.37-1.11-.56-2.3-.56-3.53 0-.54-.45-.99-.99-.99H4.19C3.65 3 3 3.24 3 3.99 3 13.28 10.73 21 20.01 21c.71 0 .99-.63.99-1.18v-3.45c0-.54-.45-.99-.99-.99z"/>
              </svg>
            </button>
            <span style="font-size: 0.75rem; color: #9CA3AF;">Decline</span>
          </div>

          <!-- Accept -->
          <div style="display: flex; flex-direction: column; align-items: center; gap: 8px;">
            <button id="tm-btn-accept" class="tm-btn tm-btn-accept">
              <svg width="24" height="24" fill="white" viewBox="0 0 24 24">
                <path d="M17 10.5V7c0-.55-.45-1-1-1H4c-.55 0-1 .45-1 1v10c0 .55.45 1 1 1h12c.55 0 1-.45 1-1v-3.5l4 4v-11l-4 4z"/>
              </svg>
            </button>
            <span style="font-size: 0.75rem; color: #fff; font-weight: 600;">Accept</span>
          </div>
        </div>
      </div>
    `;

    document.body.appendChild(overlay);

    // Accept Click handler
    document.getElementById('tm-btn-accept').onclick = async function() {
      stopRinging();
      this.disabled = true;
      this.textContent = 'Accepting...';
      try {
        const res = await fetch(`/api/v1/appointments/${appt.id}/start`, {
          method: 'PATCH',
          headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' }
        });
        if (res.ok) {
          location.href = `/consultation/${appt.id}/call`;
        }
      } catch (err) {
        console.error(err);
      } finally {
        overlay.remove();
      }
    };

    // Decline Click handler
    document.getElementById('tm-btn-decline').onclick = async function() {
      stopRinging();
      this.disabled = true;
      try {
        await fetch(`/api/v1/appointments/${appt.id}/cancel`, {
          method: 'PATCH',
          headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
          body: JSON.stringify({ reason: 'Doctor declined call' })
        });
      } catch (err) {
        console.error(err);
      } finally {
        overlay.remove();
      }
    };
  }

  // Initial connection
  connect();
})();
