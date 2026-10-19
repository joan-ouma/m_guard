#include <HardwareSerial.h>
#include <Wire.h>
#include <math.h>

// ============================================================================
// CONFIGURATION & CONSTANTS
// ============================================================================

// Hardware Pins
constexpr uint8_t PIN_SIM868_RX       = 16;
constexpr uint8_t PIN_SIM868_TX       = 17;
constexpr uint8_t PIN_SIM868_PWRKEY   = 4;
constexpr uint8_t MPU6500_I2C_ADDR    = 0x68;

// Network & Broker Configuration
const char* const BROKER_HOST         = "172.233.255.158";
constexpr uint16_t BROKER_PORT        = 42519; // Make sure this matches your deployed MQTT port
static String deviceImei              = "860123456789001"; // Dynamically queried via AT+GSN at boot
static String telemetryTopic          = "motoguard/v1/telemetry/" + deviceImei;
const char* const CELLULAR_APN        = "safaricom";

// Scheduling Intervals
constexpr uint32_t TELEMETRY_INTERVAL_MS = 10000;
constexpr uint32_t DR_INTERVAL_MS        = 100;

// Hardware Serial instance for SIM868 on UART2
HardwareSerial sim868(2);

// ============================================================================
// DATA STRUCTURES
// ============================================================================

struct ImuData {
  int16_t ax = 0, ay = 0, az = 0;
  int16_t gx = 0, gy = 0, gz = 0;
};

struct LocationResult {
  float lat = 0.0;
  float lon = 0.0;
  float speedKmh = 0.0;
  float headingDeg = 0.0;
  String locType = "none";
};

struct CellTowerInfo {
  uint16_t mcc = 639; // Kenya Mobile Country Code
  uint8_t  mnc = 2;   // Safaricom Mobile Network Code
  uint32_t lac = 0;   // Location Area Code
  uint32_t cid = 0;   // Cell Tower Identifier
  int16_t  csq = 0;   // Signal quality (0-31)
  int16_t  rssiDbm = -115; // Signal strength in dBm
};

struct CellNeighbor {
  uint32_t lac = 0;
  uint32_t cid = 0;
  int16_t  rssiDbm = -115;
};

// Global System State
static ImuData currentImu;
static CellTowerInfo currentTower;
static CellNeighbor neighborCells[7];
static int neighborCellCount = 0;
static LocationResult cachedLocation;

static float deadReckonHeadingDeg = 0.0;
static float deadReckonSpeedMps   = 0.0;
static unsigned long lastDrRunMs  = 0;
static unsigned long lastPublishMs = 0;

// Forward Declarations
bool connectTcpSocket();
void mqttSendConnectPacket();

// ============================================================================
// AT COMMAND ENGINE
// ============================================================================

bool isAtCommandDone(const String &resp, const String &waitFor) {
  if (waitFor.length() > 0) {
    return (resp.indexOf(waitFor) != -1 || resp.indexOf("ERROR") != -1 || resp.indexOf("FAIL") != -1);
  }
  return (resp.indexOf("OK") != -1 || resp.indexOf("ERROR") != -1 || 
          resp.indexOf("CONNECT") != -1 || resp.indexOf("FAIL") != -1);
}

String sendAT(const String &cmd, uint32_t timeoutMs = 2000, const String &waitFor = "") {
  while (sim868.available()) sim868.read(); // Flush stale incoming bytes
  sim868.print(cmd + "\r\n");

  String resp = "";
  uint32_t start = millis();

  while (millis() - start < timeoutMs) {
    while (sim868.available()) {
      resp += (char)sim868.read();
    }

    if (isAtCommandDone(resp, waitFor)) {
      delay(waitFor.length() > 0 ? 80 : 50);
      while (sim868.available()) resp += (char)sim868.read();
      break;
    }
    delay(10);
  }
  return resp;
}

bool waitPrompt(uint32_t timeoutMs = 2500) {
  uint32_t start = millis();
  while (millis() - start < timeoutMs) {
    while (sim868.available()) {
      if ((char)sim868.read() == '>') return true;
    }
    delay(5);
  }
  return false;
}

// ============================================================================
// HARDWARE INITIALIZATION
// ============================================================================

void initMPU() {
  Wire.begin();
  Wire.beginTransmission(MPU6500_I2C_ADDR);
  Wire.write(0x6B); // Power Management 1 register
  Wire.write(0x00); // Wake up sensor
  
  if (Wire.endTransmission() != 0) {
    Serial.println("[I2C] ERROR: MPU-6500 Not Detected.");
    return;
  }
  Serial.println("[I2C] MPU-6500 Ready.");
}

void powerOnModem() {
  Serial.println("[HW] Booting SIM868 via PWRKEY...");
  pinMode(PIN_SIM868_PWRKEY, OUTPUT);
  digitalWrite(PIN_SIM868_PWRKEY, LOW);
  delay(1200);
  pinMode(PIN_SIM868_PWRKEY, INPUT); // Release to high-Z state to prevent reset loop
  delay(5000); // Allow modem internal boot sequence to complete
}

bool readIMU(ImuData &imu) {
  Wire.beginTransmission(MPU6500_I2C_ADDR);
  Wire.write(0x3B);
  if (Wire.endTransmission(false) != 0) return false;

  Wire.requestFrom(MPU6500_I2C_ADDR, (uint8_t)14, (uint8_t)true);
  if (Wire.available() != 14) return false;

  imu.ax = (Wire.read() << 8 | Wire.read());
  imu.ay = (Wire.read() << 8 | Wire.read());
  imu.az = (Wire.read() << 8 | Wire.read());
  Wire.read(); Wire.read(); // Skip internal temperature sensor bytes
  imu.gx = (Wire.read() << 8 | Wire.read());
  imu.gy = (Wire.read() << 8 | Wire.read());
  imu.gz = (Wire.read() << 8 | Wire.read());
  return true;
}

// ============================================================================
// SENSOR PROCESSING & INERTIAL NAVIGATION (DEAD RECKONING)
// ============================================================================

void updateHeadingFromGyro(float yawRate, float dt) {
  if (abs(yawRate) <= 1.2) return; // Guard: ignore stationary sensor noise

  deadReckonHeadingDeg += yawRate * dt;
  while (deadReckonHeadingDeg < 0.0) deadReckonHeadingDeg += 360.0;
  while (deadReckonHeadingDeg >= 360.0) deadReckonHeadingDeg -= 360.0;
}

void updateSpeedFromMotion(bool inMotion, float dt) {
  if (!inMotion) {
    float coasted = deadReckonSpeedMps * 0.82;
    deadReckonSpeedMps = (coasted < 0.3) ? 0.0 : coasted;
    return;
  }
  // When in motion, decay last confirmed speed smoothly instead of fabricating artificial speed
  if (deadReckonSpeedMps > 0.0) {
    deadReckonSpeedMps *= 0.99;
  }
}

void projectDeadReckoningDisplacement(float dt) {
  if (deadReckonSpeedMps <= 0.5) return; // Guard: vehicle stationary
  if (cachedLocation.lat == 0.0 || cachedLocation.lon == 0.0) return; // Guard: no valid base fix

  float headingRad = deadReckonHeadingDeg * 0.0174532925;
  float dMeters = deadReckonSpeedMps * dt;
  float dLat = (dMeters * cos(headingRad)) / 111139.0;
  float dLon = (dMeters * sin(headingRad)) / (111139.0 * cos(cachedLocation.lat * 0.0174532925));

  cachedLocation.lat += dLat;
  cachedLocation.lon += dLon;
}

void updateDeadReckoning(const ImuData &imu, float dt) {
  if (dt <= 0.02 || dt > 1.5) return; // Guard: discard invalid time steps

  float yawRate = (float)imu.gz / 131.0;
  updateHeadingFromGyro(yawRate, dt);

  float totalG = sqrt((float)imu.ax * imu.ax + (float)imu.ay * imu.ay + (float)imu.az * imu.az) / 16384.0;
  bool inMotion = (abs(totalG - 1.0) > 0.06) || (abs(yawRate) > 2.5);
  updateSpeedFromMotion(inMotion, dt);

  projectDeadReckoningDisplacement(dt);
}

// ============================================================================
// MULTI-TIER LOCATION ENGINE
// ============================================================================

void syncGpsGroundTrack(const String &resp, int c5) {
  int c6 = resp.indexOf(',', c5 + 1);
  if (c6 == -1) return;
  int c7 = resp.indexOf(',', c6 + 1);
  if (c7 == -1) return;

  float gpsSpeedKmh = resp.substring(c5 + 1, c6).toFloat();
  if (gpsSpeedKmh <= 2.0) return; // Guard: stationary

  float gpsCourse = resp.substring(c6 + 1, c7).toFloat();
  deadReckonSpeedMps   = gpsSpeedKmh / 3.6;
  deadReckonHeadingDeg = gpsCourse;
}

bool tryGetGpsFix(LocationResult &loc) {
  String resp = sendAT("AT+CGNSINF", 1000);
  if (resp.indexOf("+CGNSINF: 0,") != -1) {
    sendAT("AT+CGNSPWR=1", 1000); // Re-enable GPS power if modem rebooted
    return false;
  }
  if (resp.indexOf("+CGNSINF: 1,1,") == -1) return false;

  int c3 = resp.indexOf(',', resp.indexOf(',', resp.indexOf(',') + 1) + 1);
  if (c3 == -1) return false;
  int c4 = resp.indexOf(',', c3 + 1);
  if (c4 == -1) return false;
  int c5 = resp.indexOf(',', c4 + 1);
  if (c5 == -1) return false;

  float lat = resp.substring(c3 + 1, c4).toFloat();
  float lon = resp.substring(c4 + 1, c5).toFloat();
  if (lat == 0.0 || lon == 0.0) return false;

  loc.lat = lat;
  loc.lon = lon;
  loc.locType = "gps";

  syncGpsGroundTrack(resp, c5);
  return true;
}

bool tryGetLbsFix(LocationResult &loc) {
  if (!isNetworkRegistered()) return false; // Guard: cellular registration required
  String resp = sendAT("AT+CIPGSMLOC=1,1", 3000);
  int startIdx = resp.indexOf("+CIPGSMLOC: 0,");
  if (startIdx == -1) return false;

  int c1 = resp.indexOf(',', startIdx);
  if (c1 == -1) return false;
  int c2 = resp.indexOf(',', c1 + 1);
  if (c2 == -1) return false;
  int c3 = resp.indexOf(',', c2 + 1);
  if (c3 == -1) return false;

  float lon = resp.substring(c1 + 1, c2).toFloat(); // Note: CIPGSMLOC outputs Longitude first
  float lat = resp.substring(c2 + 1, c3).toFloat();
  if (lat == 0.0 || lon == 0.0) return false;

  loc.lat = lat;
  loc.lon = lon;
  loc.locType = "lbs";
  return true;
}

void parseCellTowerId(const String &resp, CellTowerInfo &tower) {
  int q1 = resp.indexOf('\"');
  if (q1 == -1) return;
  int q2 = resp.indexOf('\"', q1 + 1);
  if (q2 == -1) return;
  int q3 = resp.indexOf('\"', q2 + 1);
  if (q3 == -1) return;
  int q4 = resp.indexOf('\"', q3 + 1);
  if (q4 == -1) return;

  tower.lac = strtoul(resp.substring(q1 + 1, q2).c_str(), NULL, 16);
  tower.cid = strtoul(resp.substring(q3 + 1, q4).c_str(), NULL, 16);
}

void parseSignalStrength(const String &resp, CellTowerInfo &tower) {
  int idx = resp.indexOf("+CSQ: ");
  if (idx == -1) return;

  int val = resp.substring(idx + 6).toInt();
  tower.csq = val;
  tower.rssiDbm = (val == 99 || val == 0) ? -115 : (-113 + (val * 2));
}

void updateServingCellTower(CellTowerInfo &tower) {
  parseCellTowerId(sendAT("AT+CREG?", 500), tower);
  parseSignalStrength(sendAT("AT+CSQ", 500), tower);
}

void scanNeighborCells() {
  neighborCellCount = 0;
  String resp = sendAT("AT+CENG?", 1200);
  int startIdx = resp.indexOf("+CENG:");
  if (startIdx == -1) return;

  int searchPos = 0;
  while (neighborCellCount < 7) {
    int lineStart = resp.indexOf("+CENG:", searchPos);
    if (lineStart == -1) break;

    int lineEnd = resp.indexOf('\n', lineStart);
    if (lineEnd == -1) lineEnd = resp.length();
    String line = resp.substring(lineStart, lineEnd);
    searchPos = lineEnd + 1;

    int quoteStart = line.indexOf('\"');
    int quoteEnd = line.lastIndexOf('\"');
    if (quoteStart == -1 || quoteEnd <= quoteStart) continue;

    String content = line.substring(quoteStart + 1, quoteEnd);
    int commaCount = 0;
    int prevComma = -1;
    String tokens[12];
    for (int i = 0; i <= (int)content.length(); i++) {
      bool isComma = (i < (int)content.length() && content.charAt(i) == ',');
      bool isEnd = (i == (int)content.length());
      if (!isComma && !isEnd) continue;
      if (commaCount < 12) {
        tokens[commaCount++] = content.substring(prevComma + 1, i);
      }
      prevComma = i;
    }

    if (commaCount < 7) continue;

    int rxlev = tokens[1].toInt();
    int dbm = (rxlev > 0 && rxlev <= 63) ? (rxlev - 110) : -95;
    uint32_t lac = 0;
    uint32_t cid = 0;

    bool isServing = line.startsWith("+CENG: 0");
    if (isServing && commaCount >= 10) {
      cid = strtoul(tokens[6].c_str(), NULL, 16);
      lac = strtoul(tokens[9].c_str(), NULL, 16);
    }
    if (!isServing && commaCount >= 7) {
      cid = strtoul(tokens[3].c_str(), NULL, 16);
      lac = strtoul(tokens[6].c_str(), NULL, 16);
    }

    if (cid == 0 || lac == 0) continue;

    neighborCells[neighborCellCount].lac = lac;
    neighborCells[neighborCellCount].cid = cid;
    neighborCells[neighborCellCount].rssiDbm = dbm;
    neighborCellCount++;
  }
}

LocationResult resolveBestLocation() {
  LocationResult loc;

  // Tier 1: True GNSS / Satellite Fix
  if (tryGetGpsFix(loc)) {
    cachedLocation = loc;
    return loc;
  }

  // Tier 2: Cloud LBS (AT+CIPGSMLOC)
  if (tryGetLbsFix(loc)) {
    cachedLocation = loc;
    return loc;
  }

  // Tier 3: Safaricom Cell Tower LAC/CID Lookup & Multi-Cell Scan
  updateServingCellTower(currentTower);
  scanNeighborCells();

  // Tier 4: Dead Reckoning (if moving and a valid previous fix exists)
  if (deadReckonSpeedMps > 1.0 && cachedLocation.lat != 0.0 && cachedLocation.lon != 0.0) {
    loc.lat = cachedLocation.lat;
    loc.lon = cachedLocation.lon;
    loc.locType = "dead_reckoning";
    return loc;
  }

  // Tier 5: Last Known Coordinates (if a valid previous fix exists)
  if (cachedLocation.lat != 0.0 && cachedLocation.lon != 0.0) {
    loc.lat = cachedLocation.lat;
    loc.lon = cachedLocation.lon;
    loc.locType = "last_known";
    return loc;
  }

  // No fix acquired -> Output clean zero
  loc.lat = 0.0;
  loc.lon = 0.0;
  loc.locType = "none";
  return loc;
}

// ============================================================================
// NETWORK & MQTT MANAGEMENT
// ============================================================================

void printGentleStatusReport() {
  String csqResp = sendAT("AT+CSQ", 300);
  int csq = 0;
  int csqIdx = csqResp.indexOf("+CSQ: ");
  if (csqIdx != -1) csq = csqResp.substring(csqIdx + 6).toInt();
  int rssiDbm = (csq == 99 || csq == 0) ? -115 : (-113 + (csq * 2));

  String cregResp = sendAT("AT+CREG?", 300);
  String cregStatus = "Idle (0,0)";
  if (cregResp.indexOf(",1") != -1) cregStatus = "Registered (Home)";
  if (cregResp.indexOf(",2") != -1) cregStatus = "Searching...";
  if (cregResp.indexOf(",3") != -1) cregStatus = "Registration Denied";
  if (cregResp.indexOf(",5") != -1) cregStatus = "Registered (Roaming)";

  String gpsResp = sendAT("AT+CGNSINF", 300);
  String gpsStatus = (gpsResp.indexOf("+CGNSINF: 1,1,") != -1) ? "Locked (3D Fix)" : "Searching (No Fix)";

  Serial.println("[STATUS] Signal: CSQ " + String(csq) + " (" + String(rssiDbm) + " dBm) | Cellular: " + cregStatus + " | GPS: " + gpsStatus);
}

bool isNetworkRegistered() {
  String resp = sendAT("AT+CREG?", 500);
  return (resp.indexOf(",1") != -1 || resp.indexOf(",5") != -1);
}

bool connectTcpSocket() {
  if (!isNetworkRegistered()) {
    Serial.println("[GSM] Not registered on cellular network yet.");
    return false;
  }

  Serial.println("[TCP] Reconnecting TCP Socket to Broker...");
  sendAT("AT+CIPSHUT", 1000);
  sendAT("AT+CIPMUX=0", 500);
  sendAT("AT+CIPCCFG=10,2,256,1", 500); // 10 TCP retries, 2s interval, auto-buffer
  sendAT(String("AT+CSTT=\"") + CELLULAR_APN + "\"", 1000);
  sendAT("AT+CIICR", 3000);
  sendAT("AT+CIFSR", 1000);

  String connCmd = String("AT+CIPSTART=\"TCP\",\"") + BROKER_HOST + "\"," + String(BROKER_PORT);
  String connResp = sendAT(connCmd, 8000, "CONNECT");

  if (connResp.indexOf("CONNECT OK") == -1 && connResp.indexOf("ALREADY CONNECT") == -1) {
    Serial.println("[TCP] Socket Connection Failed: " + connResp);
    return false;
  }

  Serial.println("[TCP] Socket Connected Successfully!");
  return true;
}

void queryDeviceImei() {
  String resp = sendAT("AT+GSN", 1000);
  resp.trim();
  for (int i = 0; i <= (int)resp.length() - 15; i++) {
    bool isNumeric = true;
    for (int j = 0; j < 15; j++) {
      if (!isDigit(resp.charAt(i + j))) {
        isNumeric = false;
        break;
      }
    }
    if (isNumeric) {
      deviceImei = resp.substring(i, i + 15);
      telemetryTopic = "motoguard/v1/telemetry/" + deviceImei;
      Serial.println("[MODEM] Detected Device IMEI: " + deviceImei);
      return;
    }
  }
  Serial.println("[MODEM] Using fallback IMEI: " + deviceImei);
}

void mqttSendConnectPacket() {
  Serial.println("[MQTT] Transmitting Connect Packet for " + deviceImei + "...");
  int idLen = deviceImei.length();
  int varHeaderLen = 10;
  int payloadLen = 2 + idLen;
  int remLen = varHeaderLen + payloadLen;

  sim868.print("AT+CIPSEND=");
  sim868.println(1 + 1 + remLen);
  if (!waitPrompt()) {
    Serial.println("[MQTT] Connect prompt failed.");
    return;
  }

  sim868.write(0x10); // CONNECT packet
  sim868.write((uint8_t)remLen);
  // Variable header: Protocol Name ("MQTT")
  sim868.write((uint8_t)0x00);
  sim868.write((uint8_t)0x04);
  sim868.print("MQTT");
  sim868.write((uint8_t)0x04); // Protocol Level 4 (MQTT v3.1.1)
  sim868.write((uint8_t)0x02); // Connect Flags: Clean Session
  sim868.write((uint8_t)0x00); // Keep Alive MSB
  sim868.write((uint8_t)0x3C); // Keep Alive LSB (60s)
  // Payload: Client Identifier
  sim868.write((uint8_t)(idLen >> 8));
  sim868.write((uint8_t)(idLen & 0xFF));
  sim868.print(deviceImei);
  delay(400); // Allow broker to process and acknowledge
}

void recoverTcpConnection() {
  if (!connectTcpSocket()) return; // Guard clause
  mqttSendConnectPacket();
}

bool mqttPublishPayload(const String &payload) {
  if (!isNetworkRegistered()) {
    Serial.println("[GSM] Not registered on cellular network yet.");
    return false;
  }

  int topicLen = telemetryTopic.length();
  int payloadLen = payload.length();
  int remLen = 2 + topicLen + payloadLen;

  // Encode MQTT Remaining Length (Variable Byte Integer)
  uint8_t remBytes[4];
  int remCount = 0;
  int temp = remLen;
  do {
    uint8_t d = temp % 128;
    temp /= 128;
    d |= (temp > 0) ? 128 : 0;
    remBytes[remCount++] = d;
  } while (temp > 0);

  int totalLen = 1 + remCount + 2 + topicLen + payloadLen;

  sim868.print("AT+CIPSEND=");
  sim868.println(totalLen);

  if (!waitPrompt()) {
    // Guard clause: Prompt not received, trigger reconnect
    Serial.println("[MQTT] Publish failed (Prompt not received). Re-establishing connection...");
    recoverTcpConnection();
    return false;
  }

  sim868.write(0x30); // MQTT Publish Header, QoS 0
  for (int i = 0; i < remCount; i++) sim868.write(remBytes[i]);
  sim868.write(topicLen >> 8);   // Topic length MSB
  sim868.write(topicLen & 0xFF); // Topic length LSB
  sim868.print(telemetryTopic);
  sim868.print(payload);
  Serial.println("[MQTT] Telemetry Published Successfully!");
  return true;
}

// ============================================================================
// TELEMETRY SERIALIZATION & DISPATCH
// ============================================================================

int queryBatteryPercent() {
  String resp = sendAT("AT+CBC", 500);
  int idx = resp.indexOf("+CBC: ");
  if (idx == -1) return 0;
  int c1 = resp.indexOf(',', idx);
  if (c1 == -1) return 0;
  int c2 = resp.indexOf(',', c1 + 1);
  if (c2 == -1) return 0;
  return resp.substring(c1 + 1, c2).toInt();
}

String serializeTelemetry(const LocationResult &loc, const ImuData &imu) {
  int bat = queryBatteryPercent();
  bool ign = (deadReckonSpeedMps > 0.3) || (loc.speedKmh > 1.0);

  String json = "{";
  json += "\"lat\":" + String(loc.lat, 6) + ",";
  json += "\"lng\":" + String(loc.lon, 6) + ",";
  json += "\"loc_type\":\"" + loc.locType + "\",";
  json += "\"mcc\":" + String(currentTower.mcc) + ",";
  json += "\"mnc\":" + String(currentTower.mnc) + ",";
  json += "\"lac\":" + String(currentTower.lac) + ",";
  json += "\"cid\":" + String(currentTower.cid) + ",";
  json += "\"csq\":" + String(currentTower.csq) + ",";
  json += "\"bat\":" + String(bat) + ",";
  json += "\"rssi_dbm\":" + String(currentTower.rssiDbm) + ",";
  json += "\"spd\":" + String(deadReckonSpeedMps * 3.6, 1) + ",";
  json += "\"hdg\":" + String(deadReckonHeadingDeg, 1) + ",";
  json += "\"ign\":" + String(ign ? "true" : "false") + ",";
  json += "\"ax\":" + String(imu.ax) + ",\"ay\":" + String(imu.ay) + ",\"az\":" + String(imu.az) + ",";
  json += "\"gx\":" + String(imu.gx) + ",\"gy\":" + String(imu.gy) + ",\"gz\":" + String(imu.gz);
  if (neighborCellCount > 0) {
    json += ",\"cells\":[";
    for (int i = 0; i < neighborCellCount; i++) {
      if (i > 0) json += ",";
      json += "{\"lac\":" + String(neighborCells[i].lac) + ",\"cid\":" + String(neighborCells[i].cid) + ",\"dbm\":" + String(neighborCells[i].rssiDbm) + "}";
    }
    json += "]";
  }
  json += "}";
  return json;
}

void dispatchTelemetry() {
  LocationResult loc = resolveBestLocation();
  loc.speedKmh = deadReckonSpeedMps * 3.6;
  loc.headingDeg = deadReckonHeadingDeg;

  String payload = serializeTelemetry(loc, currentImu);
  Serial.println("\n[DATA] " + payload);
  mqttPublishPayload(payload);
}

// ============================================================================
// HIGH-RATE SCHEDULER STEPS
// ============================================================================

void processDeadReckoningStep(unsigned long now) {
  if (lastDrRunMs == 0) lastDrRunMs = now;

  float dt = (now - lastDrRunMs) / 1000.0;
  if (dt < (float)DR_INTERVAL_MS / 1000.0) return; // Guard: interval not elapsed

  lastDrRunMs = now;
  if (!readIMU(currentImu)) return; // Guard: sensor read failed

  updateDeadReckoning(currentImu, dt);
}

void processPeriodicTelemetry(unsigned long now) {
  if (now - lastPublishMs < TELEMETRY_INTERVAL_MS) return; // Guard: interval not elapsed
  lastPublishMs = now;

  if (!isNetworkRegistered()) {
    printGentleStatusReport();
    return;
  }

  dispatchTelemetry();
}

// ============================================================================
// ARDUINO SETUP & LOOP
// ============================================================================

void setup() {
  Serial.begin(115200);
  initMPU();

  sim868.begin(9600, SERIAL_8N1, PIN_SIM868_RX, PIN_SIM868_TX);
  powerOnModem();
  queryDeviceImei();

  sendAT("ATE0");       // Disable command echo
  sendAT("AT+CFUN=1");  // Ensure full radio functionality is enabled
  sendAT("AT+CSCLK=0"); // Disable sleep mode (keep modem receiver at 100% sensitivity)
  sendAT("AT+CBAND=\"ALL_BAND\""); // Support both 900MHz EGSM and 1800MHz DCS
  sendAT("AT+COPS=0");  // Automatic carrier discovery
  sendAT("AT+CREG=2");  // Enable engineering registration reporting (LAC & CID)
  sendAT("AT+CGATT=1", 3000);

  // Configure GNSS Engine & A-GPS
  sendAT("AT+CGNSPWR=1", 2000); // Enable GPS power
  sendAT("AT+CGNSMOD=1,1,0,0"); // Multi-constellation: GPS + GLONASS
  sendAT("AT+CGNSEPO=1", 1000); // Enable cloud Extended Prediction Orbit assistance
  sendAT("AT+CENG=3,1", 500);   // Multi-Cell Engineering Mode

  // Wait for cellular network registration
  Serial.println("[GSM] Waiting for Safaricom registration...");
  for (int i = 0; i < 20; i++) {
    if (isNetworkRegistered()) {
      Serial.println("[GSM] Network Registered!");
      break;
    }
    delay(1000);
  }

  // Configure GPRS Bearer for Cloud LBS & Fallback
  sendAT("AT+SAPBR=3,1,\"CONTYPE\",\"GPRS\"");
  sendAT(String("AT+SAPBR=3,1,\"APN\",\"") + CELLULAR_APN + "\"");
  sendAT("AT+SAPBR=1,1", 5000);

  // Initial TCP Socket connection
  recoverTcpConnection();
}

void loop() {
  // 1. Bidirectional Serial Passthrough for live debugging
  while (sim868.available()) Serial.write(sim868.read());
  while (Serial.available()) sim868.write(Serial.read());

  unsigned long now = millis();

  // 2. High-Rate Inertial Odometry & Dead Reckoning (10 Hz)
  processDeadReckoningStep(now);

  // 3. Periodic Telemetry Dispatch (Every 10 seconds)
  processPeriodicTelemetry(now);
}
