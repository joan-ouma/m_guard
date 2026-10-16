package handler

const simulatorHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>MotoGuard — Africa's Talking USSD & SMS Simulator</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&family=Plus+Jakarta+Sans:wght@400;500;600;700;800&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg: #0b0f19;
      --card-bg: #111827;
      --card-border: #1f2937;
      --primary: #10b981;
      --primary-hover: #059669;
      --primary-glow: rgba(16, 185, 129, 0.25);
      --accent: #3b82f6;
      --danger: #ef4444;
      --warning: #f59e0b;
      --text: #f9fafb;
      --text-muted: #9ca3af;
      --phone-body: #1e293b;
      --screen-bg: #030712;
      --font-main: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, sans-serif;
      --font-mono: 'JetBrains Mono', monospace;
    }

    * { box-sizing: border-box; margin: 0; padding: 0; }

    body {
      background-color: var(--bg);
      color: var(--text);
      font-family: var(--font-main);
      min-height: 100vh;
      display: flex;
      flex-direction: column;
    }

    header {
      background: rgba(17, 24, 39, 0.8);
      backdrop-filter: blur(12px);
      border-bottom: 1px solid var(--card-border);
      padding: 16px 32px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      position: sticky;
      top: 0;
      z-index: 50;
    }

    .brand {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .brand-logo {
      width: 40px;
      height: 40px;
      border-radius: 10px;
      background: linear-gradient(135deg, #10b981, #047857);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 22px;
      box-shadow: 0 0 15px var(--primary-glow);
    }

    .brand h1 {
      font-size: 1.25rem;
      font-weight: 700;
      letter-spacing: -0.02em;
    }

    .brand span {
      color: var(--primary);
    }

    .badge {
      font-size: 0.75rem;
      padding: 4px 10px;
      border-radius: 9999px;
      font-weight: 600;
      background: rgba(16, 185, 129, 0.15);
      color: #34d399;
      border: 1px solid rgba(16, 185, 129, 0.3);
      display: inline-flex;
      align-items: center;
      gap: 6px;
    }

    .badge-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: #10b981;
      animation: pulse 2s infinite;
    }

    @keyframes pulse {
      0% { transform: scale(0.95); opacity: 0.8; }
      50% { transform: scale(1.15); opacity: 1; }
      100% { transform: scale(0.95); opacity: 0.8; }
    }

    .container {
      max-width: 1400px;
      width: 100%;
      margin: 0 auto;
      padding: 32px;
      display: grid;
      grid-template-columns: 440px 1fr;
      gap: 32px;
      flex: 1;
    }

    @media (max-width: 1024px) {
      .container { grid-template-columns: 1fr; }
    }

    /* Phone Device Mockup */
    .phone-container {
      display: flex;
      flex-direction: column;
      align-items: center;
    }

    .phone-frame {
      width: 380px;
      background: #182234;
      border: 10px solid #2d3b55;
      border-radius: 44px;
      padding: 20px 16px 28px 16px;
      box-shadow: 0 25px 60px -15px rgba(0, 0, 0, 0.8), 0 0 40px rgba(16, 185, 129, 0.1);
      position: relative;
    }

    .phone-notch {
      width: 120px;
      height: 18px;
      background: #2d3b55;
      border-radius: 0 0 14px 14px;
      margin: -20px auto 14px auto;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
    }

    .notch-speaker {
      width: 40px;
      height: 4px;
      background: #475569;
      border-radius: 2px;
    }

    .notch-camera {
      width: 6px;
      height: 6px;
      background: #0f172a;
      border-radius: 50%;
    }

    .phone-screen {
      background: var(--screen-bg);
      border-radius: 28px;
      height: 600px;
      overflow: hidden;
      display: flex;
      flex-direction: column;
      border: 1px solid rgba(255, 255, 255, 0.05);
      position: relative;
    }

    .screen-header {
      background: #0f172a;
      padding: 10px 16px;
      font-size: 0.75rem;
      display: flex;
      justify-content: space-between;
      color: #94a3b8;
      border-bottom: 1px solid #1e293b;
    }

    .tab-bar {
      display: flex;
      background: #0f172a;
      border-bottom: 1px solid #1e293b;
    }

    .tab-btn {
      flex: 1;
      padding: 10px;
      text-align: center;
      font-size: 0.82rem;
      font-weight: 600;
      color: #64748b;
      cursor: pointer;
      border: none;
      background: transparent;
      transition: all 0.2s;
    }

    .tab-btn.active {
      color: #10b981;
      border-bottom: 2px solid #10b981;
      background: rgba(16, 185, 129, 0.05);
    }

    .screen-body {
      flex: 1;
      padding: 16px;
      overflow-y: auto;
      display: flex;
      flex-direction: column;
    }

    /* USSD Modal overlay on phone screen */
    .ussd-modal {
      position: absolute;
      top: 0; left: 0; right: 0; bottom: 0;
      background: rgba(3, 7, 18, 0.95);
      backdrop-filter: blur(8px);
      padding: 24px 16px;
      display: none;
      flex-direction: column;
      justify-content: center;
      z-index: 10;
    }

    .ussd-modal.open {
      display: flex;
    }

    .ussd-box {
      background: #1e293b;
      border: 1px solid #334155;
      border-radius: 16px;
      padding: 20px;
      box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
    }

    .ussd-text {
      font-family: var(--font-mono);
      font-size: 0.85rem;
      white-space: pre-wrap;
      line-height: 1.5;
      color: #f8fafc;
      margin-bottom: 16px;
      max-height: 260px;
      overflow-y: auto;
    }

    .ussd-input {
      width: 100%;
      padding: 12px;
      background: #0f172a;
      border: 1px solid #475569;
      border-radius: 8px;
      color: white;
      font-family: var(--font-mono);
      font-size: 1rem;
      margin-bottom: 16px;
      outline: none;
    }

    .ussd-input:focus {
      border-color: #10b981;
      box-shadow: 0 0 0 2px var(--primary-glow);
    }

    .ussd-actions {
      display: flex;
      gap: 10px;
    }

    .btn {
      padding: 10px 16px;
      border-radius: 8px;
      font-weight: 600;
      font-size: 0.85rem;
      cursor: pointer;
      border: none;
      transition: all 0.2s;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      gap: 6px;
    }

    .btn-primary {
      background: var(--primary);
      color: #042f2e;
    }

    .btn-primary:hover {
      background: var(--primary-hover);
    }

    .btn-secondary {
      background: #334155;
      color: #f1f5f9;
    }

    .btn-secondary:hover {
      background: #475569;
    }

    .btn-danger {
      background: #ef4444;
      color: white;
    }

    /* Keypad */
    .dialer-screen {
      display: flex;
      flex-direction: column;
      height: 100%;
      justify-content: space-between;
    }

    .dial-display {
      text-align: center;
      padding: 20px 8px;
    }

    .dial-code {
      font-family: var(--font-mono);
      font-size: 1.8rem;
      color: #38bdf8;
      letter-spacing: 2px;
      min-height: 40px;
      word-break: break-all;
    }

    .dial-phone-sub {
      font-size: 0.78rem;
      color: #64748b;
      margin-top: 6px;
    }

    .keypad-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 12px;
      margin-bottom: 20px;
    }

    .key-btn {
      height: 52px;
      border-radius: 50%;
      background: #1e293b;
      border: 1px solid #334155;
      color: white;
      font-size: 1.25rem;
      font-weight: 600;
      cursor: pointer;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      transition: all 0.1s;
    }

    .key-btn:active {
      transform: scale(0.94);
      background: #334155;
    }

    .key-sub {
      font-size: 0.6rem;
      color: #64748b;
      font-weight: 400;
      margin-top: -2px;
    }

    .dial-actions {
      display: flex;
      justify-content: space-around;
      align-items: center;
      padding: 0 16px;
    }

    .btn-call {
      width: 60px;
      height: 60px;
      border-radius: 50%;
      background: #10b981;
      color: white;
      font-size: 1.5rem;
      border: none;
      cursor: pointer;
      box-shadow: 0 0 20px var(--primary-glow);
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s;
    }

    .btn-call:hover {
      background: #059669;
      transform: scale(1.05);
    }

    .btn-backspace {
      background: transparent;
      border: none;
      color: #94a3b8;
      font-size: 1.4rem;
      cursor: pointer;
      padding: 8px;
    }

    /* SMS Chat Thread */
    .sms-thread {
      display: flex;
      flex-direction: column;
      height: 100%;
    }

    .sms-messages {
      flex: 1;
      overflow-y: auto;
      display: flex;
      flex-direction: column;
      gap: 12px;
      padding-bottom: 12px;
    }

    .chat-bubble {
      max-width: 85%;
      padding: 10px 14px;
      border-radius: 16px;
      font-size: 0.82rem;
      line-height: 1.4;
      word-wrap: break-word;
    }

    .bubble-out {
      align-self: flex-end;
      background: #0284c7;
      color: white;
      border-bottom-right-radius: 4px;
    }

    .bubble-in {
      align-self: flex-start;
      background: #1e293b;
      color: #e2e8f0;
      border-bottom-left-radius: 4px;
      border: 1px solid #334155;
    }

    .bubble-meta {
      font-size: 0.65rem;
      margin-top: 4px;
      opacity: 0.7;
      text-align: right;
    }

    .quick-chips {
      display: flex;
      gap: 6px;
      overflow-x: auto;
      padding: 8px 0;
      scrollbar-width: none;
    }

    .chip {
      background: #1e293b;
      color: #38bdf8;
      border: 1px solid #334155;
      padding: 4px 10px;
      border-radius: 9999px;
      font-size: 0.72rem;
      font-family: var(--font-mono);
      cursor: pointer;
      white-space: nowrap;
      transition: all 0.2s;
    }

    .chip:hover {
      background: #334155;
      border-color: #38bdf8;
    }

    .sms-input-row {
      display: flex;
      gap: 8px;
      padding-top: 8px;
      border-top: 1px solid #1e293b;
    }

    .sms-field {
      flex: 1;
      background: #0f172a;
      border: 1px solid #334155;
      border-radius: 20px;
      padding: 8px 14px;
      color: white;
      font-size: 0.85rem;
      outline: none;
    }

    .sms-send-btn {
      width: 36px;
      height: 36px;
      border-radius: 50%;
      background: #10b981;
      border: none;
      color: white;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    /* Right Panel (Controls, Logs, Instructions) */
    .dashboard-panel {
      display: flex;
      flex-direction: column;
      gap: 24px;
    }

    .card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 16px;
      padding: 24px;
    }

    .card-title {
      font-size: 1.1rem;
      font-weight: 700;
      margin-bottom: 16px;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }

    .form-group {
      margin-bottom: 16px;
    }

    .form-group label {
      display: block;
      font-size: 0.8rem;
      font-weight: 600;
      color: var(--text-muted);
      margin-bottom: 6px;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .form-control {
      width: 100%;
      padding: 10px 14px;
      background: #0b0f19;
      border: 1px solid #374151;
      border-radius: 8px;
      color: white;
      font-size: 0.9rem;
      outline: none;
    }

    .form-control:focus {
      border-color: var(--primary);
    }

    .row-2 {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
    }

    .preset-pill {
      font-size: 0.75rem;
      padding: 4px 10px;
      background: #1e293b;
      border: 1px solid #334155;
      border-radius: 6px;
      cursor: pointer;
      margin-right: 6px;
      display: inline-block;
      margin-top: 6px;
      color: #94a3b8;
    }

    .preset-pill:hover {
      background: #334155;
      color: white;
    }

    .outbox-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.8rem;
    }

    .outbox-table th {
      text-align: left;
      padding: 10px;
      background: #0f172a;
      color: #94a3b8;
      border-bottom: 1px solid #1e293b;
    }

    .outbox-table td {
      padding: 10px;
      border-bottom: 1px solid #1f2937;
      color: #e5e7eb;
      font-family: var(--font-mono);
    }

    .payload-debug {
      background: #030712;
      border: 1px solid #1f2937;
      border-radius: 8px;
      padding: 12px;
      font-family: var(--font-mono);
      font-size: 0.75rem;
      color: #38bdf8;
      max-height: 180px;
      overflow-y: auto;
    }
  </style>
</head>
<body>

  <header>
    <div class="brand">
      <div class="brand-logo">🏍️</div>
      <div>
        <h1>Moto<span>Guard</span> Gateway Simulator</h1>
        <p style="font-size: 0.78rem; color: #9ca3af;">Africa's Talking Sandbox USSD (*384#) & 2-Way SMS Test Environment</p>
      </div>
    </div>
    <div>
      <span class="badge"><span class="badge-dot"></span> Sandbox Ready</span>
    </div>
  </header>

  <div class="container">
    <!-- LEFT: Mobile Device Simulator -->
    <div class="phone-container">
      <div class="phone-frame">
        <div class="phone-notch">
          <div class="notch-speaker"></div>
          <div class="notch-camera"></div>
        </div>

        <div class="phone-screen">
          <div class="screen-header">
            <span>Safaricom / Airtel</span>
            <span id="screenClock">12:00</span>
          </div>

          <div class="tab-bar">
            <button class="tab-btn active" id="tabUSSD" onclick="switchPhoneTab('ussd')">📱 USSD Dialer</button>
            <button class="tab-btn" id="tabSMS" onclick="switchPhoneTab('sms')">💬 2-Way SMS</button>
          </div>

          <div class="screen-body" id="ussdPane">
            <!-- Dialer screen -->
            <div class="dialer-screen">
              <div class="dial-display">
                <div class="dial-code" id="dialCodeDisplay">*384#</div>
                <div class="dial-phone-sub">Dialing as: <span id="displayPhoneBadge">+254712345678</span></div>
              </div>

              <div class="keypad-grid">
                <button class="key-btn" onclick="pressKey('1')">1 <span class="key-sub">&nbsp;</span></button>
                <button class="key-btn" onclick="pressKey('2')">2 <span class="key-sub">ABC</span></button>
                <button class="key-btn" onclick="pressKey('3')">3 <span class="key-sub">DEF</span></button>
                <button class="key-btn" onclick="pressKey('4')">4 <span class="key-sub">GHI</span></button>
                <button class="key-btn" onclick="pressKey('5')">5 <span class="key-sub">JKL</span></button>
                <button class="key-btn" onclick="pressKey('6')">6 <span class="key-sub">MNO</span></button>
                <button class="key-btn" onclick="pressKey('7')">7 <span class="key-sub">PQRS</span></button>
                <button class="key-btn" onclick="pressKey('8')">8 <span class="key-sub">TUV</span></button>
                <button class="key-btn" onclick="pressKey('9')">9 <span class="key-sub">WXYZ</span></button>
                <button class="key-btn" onclick="pressKey('*')">* <span class="key-sub">&nbsp;</span></button>
                <button class="key-btn" onclick="pressKey('0')">0 <span class="key-sub">+</span></button>
                <button class="key-btn" onclick="pressKey('#')"># <span class="key-sub">&nbsp;</span></button>
              </div>

              <div class="dial-actions">
                <button class="btn-call" onclick="dialUSSD()" title="Dial USSD Code">📞</button>
                <button class="btn-backspace" onclick="pressBackspace()" title="Delete">⌫</button>
              </div>
            </div>

            <!-- USSD Live Modal Overlay -->
            <div class="ussd-modal" id="ussdModal">
              <div class="ussd-box">
                <div class="ussd-text" id="ussdModalText">Connecting to MotoGuard...</div>
                <div id="ussdInputWrapper">
                  <input type="text" class="ussd-input" id="ussdModalInput" placeholder="Enter choice or PIN" autocomplete="off" onkeydown="if(event.key==='Enter') submitUSSDStep()">
                </div>
                <div class="ussd-actions">
                  <button class="btn btn-primary" id="btnUSSDSend" style="flex:1;" onclick="submitUSSDStep()">Send</button>
                  <button class="btn btn-secondary" style="flex:1;" onclick="closeUSSDSession()">Cancel</button>
                </div>
              </div>
            </div>
          </div>

          <!-- SMS screen -->
          <div class="screen-body" id="smsPane" style="display:none;">
            <div class="sms-thread">
              <div style="font-size: 0.72rem; color: #94a3b8; text-align: center; margin-bottom: 8px;">
                MotoGuard Fleet Shortcode: <strong style="color:#38bdf8;">20880</strong>
              </div>
              <div class="sms-messages" id="smsMessagesList">
                <div class="chat-bubble bubble-in">
                  Welcome to MotoGuard 2-Way SMS. Send <strong>STATUS</strong>, <strong>LOCK [PIN]</strong>, <strong>UNLOCK [PIN]</strong>, or <strong>SOS</strong>.
                  <div class="bubble-meta">System</div>
                </div>
              </div>

              <div class="quick-chips">
                <span class="chip" onclick="quickSMS('STATUS')">STATUS</span>
                <span class="chip" onclick="quickSMS('LOCK 1234')">LOCK 1234</span>
                <span class="chip" onclick="quickSMS('UNLOCK 1234')">UNLOCK 1234</span>
                <span class="chip" onclick="quickSMS('SOS')">SOS</span>
                <span class="chip" onclick="quickSMS('HELP')">HELP</span>
              </div>

              <div class="sms-input-row">
                <input type="text" class="sms-field" id="smsInputText" placeholder="Type SMS message..." onkeydown="if(event.key==='Enter') sendSMSMessage()">
                <button class="sms-send-btn" onclick="sendSMSMessage()">➤</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- RIGHT: Configuration, Webhook Info, and Outbox Logs -->
    <div class="dashboard-panel">
      <!-- Session Settings Card -->
      <div class="card">
        <div class="card-title">
          <span>⚙️ Caller Configuration & Phone Presets</span>
          <span style="font-size: 0.75rem; color: #10b981;">Live Synced</span>
        </div>

        <div class="row-2">
          <div class="form-group">
            <label>Rider / Caller Phone Number</label>
            <input type="text" class="form-control" id="cfgPhone" value="+254712345678" onchange="syncPhoneBadge()">
            <div>
              <span class="preset-pill" onclick="setPresetPhone('+254712345678')">Rider 1 (+254712345678)</span>
              <span class="preset-pill" onclick="setPresetPhone('+254798765432')">Rider 2 (+254798765432)</span>
              <span class="preset-pill" onclick="setPresetPhone('+254700000000')">Unregistered Phone</span>
            </div>
          </div>
          <div class="form-group">
            <label>USSD Service Code</label>
            <input type="text" class="form-control" id="cfgServiceCode" value="*384#">
            <div style="font-size: 0.75rem; color: #64748b; margin-top: 6px;">
              Matches your Africa's Talking Sandbox channel code
            </div>
          </div>
        </div>
      </div>

      <!-- Live Webhook Debugger Card -->
      <div class="card">
        <div class="card-title">
          <span>📡 Africa's Talking Webhook Payload Inspector</span>
          <button class="btn btn-secondary" style="padding: 4px 10px; font-size: 0.75rem;" onclick="clearDebug()">Clear</button>
        </div>
        <p style="font-size: 0.8rem; color: #94a3b8; margin-bottom: 10px;">
          When Africa's Talking calls <code style="color: #38bdf8;">/v1/ussd</code> or <code style="color: #38bdf8;">/v1/sms/incoming</code>, it transmits these parameters:
        </p>
        <div class="payload-debug" id="payloadDebugBox">// No interaction recorded yet. Dial *384# or send an SMS to observe payload.</div>
      </div>

      <!-- Outbound SMS Delivery Log Card -->
      <div class="card">
        <div class="card-title">
          <span>📤 Outbound SMS Stream (Africa's Talking SMS API)</span>
          <button class="btn btn-secondary" style="padding: 4px 10px; font-size: 0.75rem;" onclick="fetchOutbox()">Refresh Logs</button>
        </div>
        <p style="font-size: 0.8rem; color: #94a3b8; margin-bottom: 12px;">
          All outbound notifications (Lock/Unlock confirmations, SOS alerts, Security Tamper SMS) dispatched via Africa's Talking appear here:
        </p>

        <div style="max-height: 240px; overflow-y: auto;">
          <table class="outbox-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>Recipient</th>
                <th>Message Content</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody id="outboxTableBody">
              <tr>
                <td colspan="4" style="text-align: center; color: #64748b;">No outbound SMS sent yet</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>

  <script>
    // State variables
    let currentTab = 'ussd';
    let currentUSSDCode = '*384#';
    let sessionID = 'ATSession_' + Math.random().toString(36).substring(2, 10);
    let sessionTextSteps = [];
    let isSessionOpen = false;

    // Clock updater
    function updateClock() {
      const now = new Date();
      document.getElementById('screenClock').textContent = 
        now.getHours().toString().padStart(2, '0') + ':' + now.getMinutes().toString().padStart(2, '0');
    }
    setInterval(updateClock, 1000);
    updateClock();

    function syncPhoneBadge() {
      const p = document.getElementById('cfgPhone').value.trim();
      document.getElementById('displayPhoneBadge').textContent = p;
    }

    function setPresetPhone(phone) {
      document.getElementById('cfgPhone').value = phone;
      syncPhoneBadge();
    }

    function switchPhoneTab(tab) {
      currentTab = tab;
      document.getElementById('tabUSSD').classList.toggle('active', tab === 'ussd');
      document.getElementById('tabSMS').classList.toggle('active', tab === 'sms');
      document.getElementById('ussdPane').style.display = tab === 'ussd' ? 'flex' : 'none';
      document.getElementById('smsPane').style.display = tab === 'sms' ? 'flex' : 'none';
    }

    // Keypad functions
    function pressKey(k) {
      if (currentUSSDCode.length < 15) {
        currentUSSDCode += k;
        document.getElementById('dialCodeDisplay').textContent = currentUSSDCode;
      }
    }

    function pressBackspace() {
      if (currentUSSDCode.length > 0) {
        currentUSSDCode = currentUSSDCode.slice(0, -1);
        document.getElementById('dialCodeDisplay').textContent = currentUSSDCode || ' ';
      }
    }

    // USSD flow
    async function dialUSSD() {
      sessionID = 'ATSession_' + Math.random().toString(36).substring(2, 10);
      sessionTextSteps = [];
      isSessionOpen = true;

      document.getElementById('ussdModal').classList.add('open');
      document.getElementById('ussdModalText').textContent = 'Dialing ' + (currentUSSDCode || '*384#') + '...';
      document.getElementById('ussdModalInput').value = '';

      await callUSSDBackend('');
    }

    async function submitUSSDStep() {
      const inputEl = document.getElementById('ussdModalInput');
      const val = inputEl.value.trim();
      if (!val && inputEl.style.display !== 'none') return;

      sessionTextSteps.push(val);
      const combinedText = sessionTextSteps.join('*');
      inputEl.value = '';

      document.getElementById('ussdModalText').textContent = 'Processing request...';
      await callUSSDBackend(combinedText);
    }

    async function callUSSDBackend(textParam) {
      const phone = document.getElementById('cfgPhone').value.trim();
      const sCode = document.getElementById('cfgServiceCode').value.trim();

      const params = new URLSearchParams();
      params.append('sessionId', sessionID);
      params.append('serviceCode', sCode);
      params.append('phoneNumber', phone);
      params.append('text', textParam);

      logDebug('USSD Request (POST /v1/ussd)', {
        sessionId: sessionID,
        serviceCode: sCode,
        phoneNumber: phone,
        text: textParam
      });

      try {
        const resp = await fetch('/v1/ussd', {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: params.toString()
        });

        const rawText = await resp.text();
        logDebug('USSD Response (Content-Type: text/plain)', rawText);

        const modalText = document.getElementById('ussdModalText');
        const inputEl = document.getElementById('ussdModalInput');
        const btnSend = document.getElementById('btnUSSDSend');

        if (rawText.startsWith('CON ')) {
          modalText.textContent = rawText.substring(4);
          inputEl.style.display = 'block';
          btnSend.style.display = 'inline-flex';
          inputEl.focus();
        } else if (rawText.startsWith('END ')) {
          modalText.textContent = rawText.substring(4);
          inputEl.style.display = 'none';
          btnSend.style.display = 'none';
          isSessionOpen = false;
          // Refresh outbox in case SMS was dispatched
          setTimeout(fetchOutbox, 500);
        } else {
          modalText.textContent = rawText;
          inputEl.style.display = 'none';
          btnSend.style.display = 'none';
          isSessionOpen = false;
        }
      } catch (err) {
        document.getElementById('ussdModalText').textContent = 'Network error: ' + err.message;
      }
    }

    function closeUSSDSession() {
      document.getElementById('ussdModal').classList.remove('open');
      isSessionOpen = false;
      sessionTextSteps = [];
    }

    // SMS flow
    function quickSMS(cmd) {
      document.getElementById('smsInputText').value = cmd;
      sendSMSMessage();
    }

    async function sendSMSMessage() {
      const field = document.getElementById('smsInputText');
      const text = field.value.trim();
      if (!text) return;

      const phone = document.getElementById('cfgPhone').value.trim();
      field.value = '';

      // Append Rider outgoing bubble
      appendSMSBubble(text, 'out', 'Rider (' + phone + ')');

      const params = new URLSearchParams();
      params.append('from', phone);
      params.append('to', '20880');
      params.append('text', text);
      params.append('id', 'AT_SMS_' + Date.now());

      logDebug('Incoming SMS Webhook (POST /v1/sms/incoming)', {
        from: phone,
        to: '20880',
        text: text
      });

      try {
        const resp = await fetch('/v1/sms/incoming', {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: params.toString()
        });

        const data = await resp.json();
        logDebug('SMS Webhook Response', data);

        if (data.reply) {
          appendSMSBubble(data.reply, 'in', 'MotoGuard 20880');
        }

        setTimeout(fetchOutbox, 500);
      } catch (err) {
        appendSMSBubble('Error sending SMS: ' + err.message, 'in', 'Error');
      }
    }

    function appendSMSBubble(content, type, sender) {
      const list = document.getElementById('smsMessagesList');
      const bubble = document.createElement('div');
      bubble.className = 'chat-bubble ' + (type === 'out' ? 'bubble-out' : 'bubble-in');

      const now = new Date();
      const timeStr = now.getHours().toString().padStart(2, '0') + ':' + now.getMinutes().toString().padStart(2, '0');

      bubble.innerHTML = content.replace(/\n/g, '<br>') + '<div class="bubble-meta">' + sender + ' • ' + timeStr + '</div>';
      list.appendChild(bubble);
      list.scrollTop = list.scrollHeight;
    }

    // Debug & Outbox functions
    function logDebug(title, data) {
      const box = document.getElementById('payloadDebugBox');
      const ts = new Date().toLocaleTimeString();
      const formatted = typeof data === 'string' ? data : JSON.stringify(data, null, 2);
      box.innerHTML = '<strong>[' + ts + '] ' + title + '</strong>\n' + formatted + '\n\n' + box.innerHTML;
    }

    function clearDebug() {
      document.getElementById('payloadDebugBox').textContent = '// Debugger cleared.';
    }

    async function fetchOutbox() {
      try {
        const resp = await fetch('/v1/sms/outbox');
        const json = await resp.json();
        const tbody = document.getElementById('outboxTableBody');
        tbody.innerHTML = '';

        if (!json.data || json.data.length === 0) {
          tbody.innerHTML = '<tr><td colspan="4" style="text-align: center; color: #64748b;">No outbound SMS recorded yet</td></tr>';
          return;
        }

        json.data.forEach(item => {
          const row = document.createElement('tr');
          const timeStr = new Date(item.created_at).toLocaleTimeString();
          row.innerHTML = '<td>' + timeStr + '</td>' +
                          '<td style="color:#38bdf8;">' + item.to + '</td>' +
                          '<td>' + item.message + '</td>' +
                          '<td><span style="color:#10b981; font-weight:600;">' + item.status + '</span></td>';
          tbody.appendChild(row);
        });
      } catch (err) {
        console.error('Fetch outbox error:', err);
      }
    }

    // Initial load
    fetchOutbox();
    setInterval(fetchOutbox, 5000);
  </script>
</body>
</html>
`
