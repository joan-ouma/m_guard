import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../../../environments/environment';

const STORAGE_KEY_SNS_ENDPOINT = 'motorguard_sns_endpoint_arn';

interface SnsRegisterResponse {
  data: {
    endpoint_arn: string;
    message: string;
  };
}

@Injectable({
  providedIn: 'root',
})
export class NotificationService {
  private readonly http = inject(HttpClient);
  private readonly apiBase = environment.apiBaseUrl;

  readonly isSupported = signal<boolean>(
    typeof window !== 'undefined' && 'Notification' in window
  );
  readonly permission = signal<NotificationPermission>(this.getInitialPermission());
  readonly isSnsSubscribed = signal<boolean>(this.getStoredEndpoint() !== null);
  readonly snsEndpointArn = signal<string | null>(this.getStoredEndpoint());
  readonly isRegistering = signal<boolean>(false);
  readonly statusMessage = signal<string | null>(null);

  private getInitialPermission(): NotificationPermission {
    if (typeof window !== 'undefined' && 'Notification' in window) {
      return Notification.permission;
    }
    return 'denied';
  }

  private getStoredEndpoint(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem(STORAGE_KEY_SNS_ENDPOINT);
  }

  async requestPermission(): Promise<boolean> {
    if (!this.isSupported()) return false;

    try {
      const result = await Notification.requestPermission();
      this.permission.set(result);
      return result === 'granted';
    } catch {
      return false;
    }
  }

  /**
   * Registers the browser / PWA Service Worker with backend AWS SNS
   */
  async registerWithSNS(phoneNumber?: string): Promise<boolean> {
    if (!this.isSupported()) return false;
    this.isRegistering.set(true);
    this.statusMessage.set('Requesting permission...');

    try {
      const granted = await this.requestPermission();
      if (!granted) {
        this.statusMessage.set('Notification permission was denied');
        return false;
      }

      this.statusMessage.set('Obtaining device push token...');
      let pushToken = '';

      if ('serviceWorker' in navigator) {
        const reg = await navigator.serviceWorker.ready;
        // Check for existing or new PushManager subscription
        if ('pushManager' in reg) {
          let sub = await reg.pushManager.getSubscription();
          if (!sub) {
            // Subscribe using public key or anonymous browser push
            try {
              sub = await reg.pushManager.subscribe({
                userVisibleOnly: true,
                applicationServerKey: this.urlBase64ToUint8Array(
                  // Fallback public VAPID key for browser push registration
                  'BEl62iUYgUivxIkv69yViEuiBIa-Ib9-SkvMeAtA3LFgDzkrxZJjSgSnfckjBJuBkr3qBUYIHBQFLXYp5Nksh8U'
                ) as unknown as BufferSource,
              });
            } catch (subErr) {
              console.warn('PushManager subscription warning, generating device token:', subErr);
            }
          }

          if (sub) {
            pushToken = JSON.stringify(sub);
          }
        }
      }

      // If browser push subscription couldn't be generated, generate unique browser device token
      if (!pushToken) {
        let clientID = localStorage.getItem('motorguard_client_device_id');
        if (!clientID) {
          clientID = 'web_' + Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
          localStorage.setItem('motorguard_client_device_id', clientID);
        }
        pushToken = clientID;
      }

      this.statusMessage.set('Registering with AWS SNS...');
      const response = await firstValueFrom(
        this.http.post<SnsRegisterResponse>(`${this.apiBase}/v1/notifications/sns-endpoint`, {
          token: pushToken,
          platform: 'GCM',
          phone_number: phoneNumber || '',
        })
      );

      const endpointArn = response?.data?.endpoint_arn || '';
      if (endpointArn) {
        localStorage.setItem(STORAGE_KEY_SNS_ENDPOINT, endpointArn);
        this.snsEndpointArn.set(endpointArn);
        this.isSnsSubscribed.set(true);
        this.statusMessage.set('SNS alerts enabled!');

        // Display confirmation push notification
        await this.notify(
          'MotorGuard Push Alerts Enabled',
          'You will receive instant lockscreen alerts for geofence breaches, tampering, and theft.'
        );
        return true;
      }

      return false;
    } catch (err: any) {
      console.error('SNS registration failed:', err);
      this.statusMessage.set(err?.error?.message || 'Failed to connect with AWS SNS');
      return false;
    } finally {
      this.isRegistering.set(false);
    }
  }

  /**
   * Unregisters device endpoint from AWS SNS
   */
  async unregisterFromSNS(): Promise<void> {
    const endpointArn = this.snsEndpointArn();
    if (endpointArn) {
      try {
        await firstValueFrom(
          this.http.delete(`${this.apiBase}/v1/notifications/sns-endpoint`, {
            params: { endpoint_arn: endpointArn },
          })
        );
      } catch (err) {
        console.warn('Error during SNS unregister:', err);
      }
    }

    localStorage.removeItem(STORAGE_KEY_SNS_ENDPOINT);
    this.snsEndpointArn.set(null);
    this.isSnsSubscribed.set(false);
    this.statusMessage.set('SNS notifications disabled');
  }

  /**
   * Triggers a test push and SMS alert via AWS SNS backend
   */
  async sendTestSNSAlert(phoneNumber?: string, bikeId?: string): Promise<boolean> {
    try {
      await firstValueFrom(
        this.http.post(`${this.apiBase}/v1/notifications/test`, {
          title: '🚨 MotorGuard Test Alert',
          body: 'Security dispatch verified via AWS SNS. All alert channels active.',
          phone_number: phoneNumber || '',
          bike_id: bikeId || '',
        })
      );
      return true;
    } catch (err) {
      console.error('Failed to trigger test SNS alert:', err);
      return false;
    }
  }

  async notify(title: string, body?: string, url?: string): Promise<void> {
    if (!this.isSupported()) return;

    if (this.permission() !== 'granted') {
      const granted = await this.requestPermission();
      if (!granted) return;
    }

    const options: NotificationOptions = {
      body: body || 'Security event detected',
      icon: '/favicon.svg',
      badge: '/favicon.svg',
      tag: `mg-alert-${Date.now()}`,
      data: { url: url || '/' },
    };

    if ('serviceWorker' in navigator) {
      try {
        const reg = await navigator.serviceWorker.ready;
        if (reg && reg.showNotification) {
          await reg.showNotification(title, options);
          return;
        }
      } catch (err) {
        console.warn('SW notification fallback to Notification constructor:', err);
      }
    }

    try {
      new Notification(title, options);
    } catch (e) {
      console.warn('Could not display system notification:', e);
    }
  }

  private urlBase64ToUint8Array(base64String: string): Uint8Array {
    const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
    const base64 = (base64String + padding).replace(/\-/g, '+').replace(/_/g, '/');
    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);
    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
  }
}
