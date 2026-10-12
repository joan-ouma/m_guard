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
   cp .env.example .env
   ```
3. Start the services using Docker Compose:
   ```bash
   docker-compose up -d
   ```
4. The dashboard will be available at `http://localhost:4200` (or as configured).

## License 
This project is licensed under the MIT License - see the LICENSE file for details.
