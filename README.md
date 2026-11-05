# MotorGuard 

**MotorGuard** is an ESP32-based stealth tracking and evidence system specifically designed for Kenyan boda bodas (motorcycle taxis). It uses motion-triggered telemetry and hashed, timestamped records to create verifiable proof of theft. Built by **the MotorGuard Team**, MotorGuard aims to ensure dignity, security, and recourse for informal transport workers in Kenya.

## Overview
Boda bodas are a vital part of Kenya's economy, but riders face immense risks, including motorcycle theft, hijacking, and loss of livelihood. Traditional trackers are easily found and destroyed. MotorGuard offers a **stealthy, tamper-proof, and low-cost** alternative that not only tracks the motorcycle but also provides verifiable cryptographic proof of its location and movement to aid in recovery and legal recourse.

## Key Features 
- **Stealth Tracking**: Motion-triggered telemetry that activates when unauthorized movement is detected.
- **Cryptographic Evidence**: Hashed and timestamped location records ensuring that data cannot be tampered with.
- **ESP32-Based Hardware**: Low-power, cost-effective, and easy to hide on the motorcycle.
- **MQTT Integration**: Reliable, lightweight messaging for poor network coverage areas.
- **Full-Stack Application**: Includes a robust backend for data ingestion and a frontend dashboard for real-time monitoring.

## Tech Stack 
- **Hardware**: ESP32 microcontroller with GPS and GSM modules
- **Backend**: Go / Node.js (with MQTT broker integration)
- **Frontend**: Modern Web App (Angular/React)
- **Infrastructure**: Docker & Docker Compose for easy deployment

## Getting Started 
### Prerequisites
- Docker & Docker Compose
- Node.js (for frontend development)
- Go (for backend development)

### Running Locally
1. Clone this repository.
2. Ensure you have copied the environment variables:
   ```bash
   cp backend/.env.example .env
   ```
3. Start the services using Docker Compose:
   ```bash
   docker-compose up -d
   ```
4. The dashboard will be available at `http://localhost:4200` (or as configured).

## Two-way SMS (Africa's Talking)

The backend accepts Africa's Talking inbound SMS callbacks at:

`https://<your-hosted-domain>/sms/africastalking`

There is no SMS text box in the web app. Riders use an ordinary phone's SMS app and send a message **to the shared shortcode `20384`**. The account uses username `buildathon-voice` and the command prefix `kamilimu ignition`.

1. Copy `backend/.env.example` to the root `.env` file and configure `AT_USERNAME`, `AT_API_KEY`, and `AT_SMS_FROM`. Set `SMS_ROUTING_KEYWORD=kamilimu` and `SMS_PROJECT_KEYWORD=ignition`. `.env` is ignored by Git; do not commit real provider credentials.
2. Deploy the backend to a public HTTPS host and give the gateway administrator the callback URL `https://<your-hosted-domain>/sms/africastalking` to register for the Ignition project.
3. From a phone, text `kamilimu ignition police Rongai` to `20384`. The service should reply with the first matching police-station entry. If the gateway strips the routing prefix before forwarding, `police Rongai` also works; `HELP` returns usage guidance.

For a local callback test, start the backend with the SMS environment variables configured, then POST a simulated callback from PowerShell:

```powershell
Invoke-WebRequest -Method Post `
  -Uri http://localhost:8080/sms/africastalking `
  -ContentType "application/x-www-form-urlencoded" `
  -Body @{ from = "+2547XXXXXXXX"; text = "kamilimu ignition police Rongai" }
```

Replace the sender with a phone number you control. This local test still calls Africa's Talking to send a real reply SMS, but it does not test the shared gateway's inbound routing. To test end-to-end, use the public registered callback and a controlled phone. A successful local test does not confirm shortcode permissions or live delivery.

The station lookup uses the area named by the rider, not GPS: an inbound SMS does not provide the sender's location, and the current station list does not calculate geographic distance. Verify directory phone numbers before riders rely on them for emergencies.

## License 
This project is licensed under the MIT License - see the LICENSE file for details.
