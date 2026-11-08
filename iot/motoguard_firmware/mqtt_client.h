#pragma once
#include <Arduino.h>
#include "config.h"
#include "modem.h"

namespace mqtt {

inline bool connectTcpSocket() {
  if (!modem::isNetworkRegistered()) {
    Serial.println("[GSM] Not registered on cellular network yet.");
    return false;
  }

  Serial.println("[TCP] Reconnecting TCP/SSL Socket to Broker...");
  modem::sendAT("AT+CIPSHUT", 1000);
  modem::sendAT("AT+CIPMUX=0", 500);
  modem::sendAT("AT+CIPRXGET=0", 500);
  modem::sendAT("AT+CIPHEAD=1", 500);
  modem::sendAT("AT+CIPCCFG=10,2,256,1", 500);
  modem::sendAT(String("AT+CSTT=\"") + CELLULAR_APN + "\"", 1000);
  modem::sendAT("AT+CIICR", 3000);
  modem::sendAT("AT+CIFSR", 1000);

  if (USE_SSL) {
    modem::sendAT("AT+CIPSSL=1", 1000);
  } else {
    modem::sendAT("AT+CIPSSL=0", 1000);
  }

  String connCmd = String("AT+CIPSTART=\"TCP\",\"") + BROKER_HOST + "\"," + String(BROKER_PORT);
  String connResp = modem::sendAT(connCmd, 15000, "CONNECT");

  if (connResp.indexOf("CONNECT OK") == -1 && connResp.indexOf("ALREADY CONNECT") == -1) {
    Serial.println("[TCP] Socket Connection Failed: " + connResp);
    return false;
  }

  Serial.println("[TCP] Socket Connected Successfully!");
  return true;
}

inline void sendConnectPacket() {
  Serial.println("[MQTT] Transmitting Authenticated Connect Packet to Broker...");

  uint16_t clientIdLen = strlen(CLIENT_ID);
  uint16_t userLen     = strlen(MQTT_USER);
  uint16_t passLen     = strlen(MQTT_PASSWORD);

  int remLen = 10 + (2 + clientIdLen) + (2 + userLen) + (2 + passLen);

  uint8_t remBytes[4];
  int remCount = 0;
  int temp = remLen;
  do {
    uint8_t d = temp % 128;
    temp /= 128;
    if (temp > 0) d |= 128;
    remBytes[remCount++] = d;
  } while (temp > 0);

  int totalLen = 1 + remCount + remLen;

  modem::flushInput();

  modem::sim868.print("AT+CIPSEND=");
  modem::sim868.println(totalLen);

  if (!modem::waitPrompt()) {
    Serial.println("[MQTT] Connect prompt failed.");
    return;
  }

  modem::sim868.write((uint8_t)0x10);

  for (int i = 0; i < remCount; i++) {
    modem::sim868.write(remBytes[i]);
  }

  uint8_t varHeader[] = {
    0x00, 0x04, 'M', 'Q', 'T', 'T',
    0x04, 0xC2, 0x00, 0xB4
  };
  modem::sim868.write(varHeader, sizeof(varHeader));

  modem::sim868.write((uint8_t)(clientIdLen >> 8));
  modem::sim868.write((uint8_t)(clientIdLen & 0xFF));
  modem::sim868.print(CLIENT_ID);

  modem::sim868.write((uint8_t)(userLen >> 8));
  modem::sim868.write((uint8_t)(userLen & 0xFF));
  modem::sim868.print(MQTT_USER);

  modem::sim868.write((uint8_t)(passLen >> 8));
  modem::sim868.write((uint8_t)(passLen & 0xFF));
  modem::sim868.print(MQTT_PASSWORD);

  delay(400);
}

inline bool subscribe(const char* topic) {
  int topicLen = strlen(topic);
  int remLen = 2 + 2 + topicLen + 1;
  int totalLen = 1 + 1 + remLen;

  modem::flushInput();

  modem::sim868.print("AT+CIPSEND=");
  modem::sim868.println(totalLen);

  if (!modem::waitPrompt()) {
    Serial.println("[MQTT] Subscribe failed (prompt not received).");
    return false;
  }

  modem::sim868.write(0x82);
  modem::sim868.write(remLen);
  modem::sim868.write(0x00);
  modem::sim868.write(0x01);
  modem::sim868.write(topicLen >> 8);
  modem::sim868.write(topicLen & 0xFF);
  modem::sim868.print(topic);
  modem::sim868.write(0x01);
  return true;
}

inline void recoverTcpConnection() {
  if (!connectTcpSocket()) return;
  sendConnectPacket();
  delay(500);
  if (subscribe(COMMAND_TOPIC)) Serial.println("[MQTT] Subscribed to command topic.");
}

inline bool publish(const char* topic, const String &payload) {
  int topicLen = strlen(topic);
  int payloadLen = payload.length();
  int remLen = 2 + topicLen + payloadLen;

  uint8_t remBytes[4];
  int remCount = 0;
  int temp = remLen;
  do {
    uint8_t d = temp % 128;
    temp /= 128;
    if (temp > 0) d |= 128;
    remBytes[remCount++] = d;
  } while (temp > 0);

  int totalLen = 1 + remCount + 2 + topicLen + payloadLen;

  modem::flushInput();

  modem::sim868.print("AT+CIPSEND=");
  modem::sim868.println(totalLen);

  if (!modem::waitPrompt()) {
    Serial.println("[MQTT] Publish failed (prompt not received). Re-establishing connection...");
    recoverTcpConnection();
    return false;
  }

  modem::sim868.write(0x30);
  for (int i = 0; i < remCount; i++) modem::sim868.write(remBytes[i]);
  modem::sim868.write(topicLen >> 8);
  modem::sim868.write(topicLen & 0xFF);
  modem::sim868.print(topic);
  modem::sim868.print(payload);
  return true;
}

// QoS 1 publish: the broker answers with a PUBACK carrying the same packet id.
inline bool publishQos1(const char* topic, const String &payload, uint16_t packetId) {
  int topicLen = strlen(topic);
  int remLen = 2 + topicLen + 2 + payload.length();

  uint8_t remBytes[4];
  int remCount = 0;
  int temp = remLen;
  do {
    uint8_t d = temp % 128;
    temp /= 128;
    if (temp > 0) d |= 128;
    remBytes[remCount++] = d;
  } while (temp > 0);

  int totalLen = 1 + remCount + remLen;

  modem::flushInput();

  modem::sim868.print("AT+CIPSEND=");
  modem::sim868.println(totalLen);

  if (!modem::waitPrompt()) {
    Serial.println("[MQTT] QoS1 publish failed (prompt not received). Re-establishing connection...");
    recoverTcpConnection();
    return false;
  }

  modem::sim868.write(0x32);
  for (int i = 0; i < remCount; i++) modem::sim868.write(remBytes[i]);
  modem::sim868.write(topicLen >> 8);
  modem::sim868.write(topicLen & 0xFF);
  modem::sim868.print(topic);
  modem::sim868.write(packetId >> 8);
  modem::sim868.write(packetId & 0xFF);
  modem::sim868.print(payload);
  return true;
}

// Reads the modem until the PUBACK for packetId arrives or the timeout expires.
inline bool waitPubAck(uint16_t packetId, uint32_t timeoutMs) {
  pubAckSeen = false;
  uint32_t start = millis();
  while (millis() - start < timeoutMs) {
    while (modem::readChar() != -1) {}
    if (pubAckSeen && lastPubAckId == packetId) return true;
    delay(10);
  }
  return false;
}

inline bool publishTelemetry(const String &payload) {
  if (!modem::isNetworkRegistered()) {
    Serial.println("[GSM] Not registered on cellular network yet.");
    return false;
  }
  bool ok = publish(TELEMETRY_TOPIC, payload);
  if (ok) Serial.println("[MQTT] Telemetry Published Successfully!");
  return ok;
}

inline bool publishAlert(const String &payload) {
  if (!modem::isNetworkRegistered()) return false;
  bool ok = publish(ALERT_TOPIC, payload);
  if (ok) Serial.println("[ALERT] Burglary Alert Published via MQTT!");
  return ok;
}

// Inbound: the SIM868 delivers received TCP data as "+RECEIVE,<len>:" followed by <len> raw bytes.
inline String rxHeader;
inline bool   rxInBody   = false;
inline size_t rxExpected = 0;
inline size_t rxCount    = 0;
inline uint8_t rxBuf[256];

struct Command {
  String   payload;
  uint16_t packetId;
};

inline uint16_t nextPacketId = 1;
inline uint16_t lastPubAckId = 0;
inline bool     pubAckSeen   = false;

constexpr uint8_t CMD_QUEUE_SIZE = 4;
inline Command cmdQueue[CMD_QUEUE_SIZE];
inline uint8_t cmdHead  = 0;
inline uint8_t cmdCount = 0;

inline void handlePublish(const uint8_t* pkt, size_t len) {
  if (len >= 4 && (pkt[0] & 0xF0) == 0x40) {
    lastPubAckId = (pkt[2] << 8) | pkt[3];
    pubAckSeen = true;
    return;
  }
  if (len < 4 || (pkt[0] & 0xF0) != 0x30) return;

  size_t i = 1;
  while (i < len && (pkt[i] & 0x80)) i++;
  i++;
  if (i + 2 > len) return;

  uint16_t topicLen = (pkt[i] << 8) | pkt[i + 1];
  i += 2;
  if (i + topicLen > len) return;

  String topic((const char*)pkt + i, topicLen);
  i += topicLen;

  int qos = (pkt[0] >> 1) & 0x03;
  uint16_t packetId = 0;
  if (qos > 0) {
    if (i + 2 > len) return;
    packetId = (pkt[i] << 8) | pkt[i + 1];
    i += 2;
  }

  // Not acknowledged when the queue is full, so the broker redelivers it later.
  if (topic.equals(COMMAND_TOPIC) && cmdCount < CMD_QUEUE_SIZE) {
    uint8_t slot = (cmdHead + cmdCount) % CMD_QUEUE_SIZE;
    cmdQueue[slot] = {String((const char*)pkt + i, len - i), packetId};
    cmdCount++;
  }
}

inline size_t lastNumber(const String &s) {
  size_t value = 0, current = 0;
  bool inNumber = false;
  for (unsigned int i = 0; i < s.length(); i++) {
    char ch = s.charAt(i);
    if (ch >= '0' && ch <= '9') {
      current = current * 10 + (ch - '0');
      inNumber = true;
    } else if (inNumber) {
      value = current;
      current = 0;
      inNumber = false;
    }
  }
  return inNumber ? current : value;
}

// Returns true when the byte is packet data, so callers never read it as AT text.
// Accepts "+IPD,<len>:" (AT+CIPHEAD=1) and the "+RECEIVE" variants.
inline bool feedInbound(char c) {
  if (rxInBody) {
    rxBuf[rxCount++] = (uint8_t)c;
    if (rxCount == rxExpected) {
      rxInBody = false;
      handlePublish(rxBuf, rxCount);
    }
    return true;
  }

  if (c != ':' && c != '\n') {
    rxHeader += c;
    if (rxHeader.length() > 40) rxHeader.remove(0, 1);
    return false;
  }

  int idx = rxHeader.lastIndexOf("+IPD");
  if (idx == -1) idx = rxHeader.lastIndexOf("+RECEIVE");
  if (idx != -1) {
    size_t len = lastNumber(rxHeader.substring(idx));
    if (len > 0 && len <= sizeof(rxBuf)) {
      rxExpected = len;
      rxCount = 0;
      rxInBody = true;
    }
  }
  rxHeader = "";
  return false;
}

inline bool takeCommand(String &payload, uint16_t &packetId) {
  if (cmdCount == 0) return false;
  payload  = cmdQueue[cmdHead].payload;
  packetId = cmdQueue[cmdHead].packetId;
  cmdHead = (cmdHead + 1) % CMD_QUEUE_SIZE;
  cmdCount--;
  return true;
}

inline bool sendPuback(uint16_t packetId) {
  modem::flushInput();

  modem::sim868.print("AT+CIPSEND=");
  modem::sim868.println(4);

  if (!modem::waitPrompt()) {
    Serial.println("[MQTT] PUBACK failed (prompt not received).");
    return false;
  }

  modem::sim868.write(0x40);
  modem::sim868.write(0x02);
  modem::sim868.write(packetId >> 8);
  modem::sim868.write(packetId & 0xFF);
  return true;
}

}

inline bool connectTcpSocket() { return mqtt::connectTcpSocket(); }
inline void mqttSendConnectPacket() { mqtt::sendConnectPacket(); }
inline void recoverTcpConnection() { mqtt::recoverTcpConnection(); }
inline bool mqttPublish(const char* topic, const String &payload) { return mqtt::publish(topic, payload); }
inline bool mqttPublishPayload(const String &payload) { return mqtt::publishTelemetry(payload); }
inline bool mqttPublishAlert(const String &payload) { return mqtt::publishAlert(payload); }
