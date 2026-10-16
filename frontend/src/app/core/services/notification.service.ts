import { Injectable, inject, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { firstValueFrom } from 'rxjs';
import { environment } from '../../../environments/environment';
import Swal from 'sweetalert2';

const STORAGE_KEY_SNS_ENDPOINT = 'motorguard_sns_endpoint_arn';

interface SnsRegisterResponse {
  data: {
    endpoint_arn: string;
    message: string;
  };
}

function urlBase64ToUint8Array(base64String: string): BufferSource {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/\-/g, '+').replace(/_/g, '/');
  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);
  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray as unknown as BufferSource;
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
   * Registers the browser / PWA Service Worker with backend notification service
   */
  async registerWithSNS(phoneNumber?: string): Promise<boolean> {
    if (!this.isSupported()) {
      Swal.fire({
        icon: 'info',
        title: 'Not Supported',
        text: 'Push notifications are not supported on this browser or platform.',
        confirmButtonColor: '#09090b',
      });
      return false;
    }

    if (typeof Notification !== 'undefined' && Notification.permission === 'denied') {
      Swal.fire({
        icon: 'warning',
        title: 'Permission Blocked',
        text: 'Notifications are blocked in your browser. Click the lock/settings icon in the address bar to set Notifications to "Allow", then try again.',
        confirmButtonColor: '#09090b',
      });
      return false;
    }

    this.isRegistering.set(true);
    this.statusMessage.set('Requesting permission...');

    try {
      const granted = await this.requestPermission();
      if (!granted) {
        Swal.fire({
          icon: 'warning',
          title: 'Permission Required',
          text: 'Notification permission was not granted. Please allow notifications when prompted.',
          confirmButtonColor: '#09090b',
        });
        return false;
      }

      this.statusMessage.set('Obtaining device token...');
      let pushToken = '';

      // Attempt to obtain genuine browser PushManager subscription
      if (typeof navigator !== 'undefined' && 'serviceWorker' in navigator && 'PushManager' in window) {
        try {
          const reg = await navigator.serviceWorker.ready;
          let sub = await reg.pushManager.getSubscription();

          if (!sub && environment.vapidPublicKey) {
            sub = await reg.pushManager.subscribe({
              userVisibleOnly: true,
              applicationServerKey: urlBase64ToUint8Array(environment.vapidPublicKey),
            });
          }

          if (sub) {
            // Chrome / Edge FCM endpoint: https://fcm.googleapis.com/fcm/send/<DEVICE_TOKEN>
            const endpointParts = sub.endpoint.split('/');
            pushToken = endpointParts[endpointParts.length - 1];
          }
        } catch (subErr) {
          console.warn('Native PushManager subscription failed or VAPID key not configured:', subErr);
        }
      }

      // Fallback to client device ID if pushManager is not configured
      if (!pushToken) {
        let clientID = localStorage.getItem('motorguard_client_device_id');
        if (!clientID) {
          clientID = 'web_' + Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
          localStorage.setItem('motorguard_client_device_id', clientID);
        }
        pushToken = clientID;
      }

      this.statusMessage.set('Enabling push notifications...');
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
        this.statusMessage.set('Push alerts enabled!');

        Swal.fire({
          toast: true,
          position: 'top-end',
          icon: 'success',
          title: 'Push Notifications Enabled',
          showConfirmButton: false,
          timer: 3000,
          timerProgressBar: true,
        });

        // Display confirmation push notification
        await this.notify(
          'MotorGuard Push Alerts Enabled',
          'You will receive instant lockscreen alerts for geofence breaches, tampering, and theft.'
        );
        return true;
      }

      return false;
    } catch (err: any) {
      console.error('Notification registration failed:', err);
      const detail = err?.error?.detail || err?.error?.message || 'Failed to enable push notifications';
      this.statusMessage.set(detail);
      Swal.fire({
        icon: 'error',
        title: 'Registration Failed',
        text: detail,
        confirmButtonColor: '#09090b',
      });
      return false;
    } finally {
      this.isRegistering.set(false);
    }
  }

  /**
   * Unregisters device endpoint from backend notification service
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
        console.warn('Error during notification unregister:', err);
      }
    }

    localStorage.removeItem(STORAGE_KEY_SNS_ENDPOINT);
    this.snsEndpointArn.set(null);
    this.isSnsSubscribed.set(false);
    this.statusMessage.set('Push notifications disabled');
  }

  /**
   * Triggers a test push notification
   */
  async sendTestSNSAlert(phoneNumber?: string, bikeId?: string): Promise<boolean> {
    try {
      await firstValueFrom(
        this.http.post(`${this.apiBase}/v1/notifications/test`, {
          title: '🚨 MotorGuard Test Alert',
          body: 'Security dispatch verified. Push notification channels active.',
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

    // 1. Direct browser Notification constructor (instant on desktop)
    if (typeof Notification !== 'undefined' && Notification.permission === 'granted') {
      try {
        new Notification(title, options);
        return;
      } catch {
        // Mobile browsers may require ServiceWorker showNotification
      }
    }

    // 2. Service Worker fallback (Mobile PWA)
    if (typeof navigator !== 'undefined' && 'serviceWorker' in navigator) {
      try {
        const reg = await Promise.race([
          navigator.serviceWorker.ready,
          new Promise<null>((resolve) => setTimeout(() => resolve(null), 500)),
        ]);
        if (reg && reg.showNotification) {
          await reg.showNotification(title, options);
          return;
        }
      } catch (err) {
        console.warn('Could not display system notification:', err);
      }
    }
  }
}
