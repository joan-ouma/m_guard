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
  readonly isSnsSubscribed = signal<boolean>(this.getInitialSubscriptionStatus());
  readonly snsEndpointArn = signal<string | null>(this.getStoredEndpoint());
  readonly isRegistering = signal<boolean>(false);
  readonly statusMessage = signal<string | null>(null);

  constructor() {
    this.initPermissionWatcher();
  }

  private getInitialPermission(): NotificationPermission {
    if (typeof window !== 'undefined' && 'Notification' in window) {
      return Notification.permission;
    }
    return 'denied';
  }

  private getInitialSubscriptionStatus(): boolean {
    if (typeof window === 'undefined' || !('Notification' in window)) {
      return false;
    }
    if (Notification.permission !== 'granted') {
      return false;
    }
    return this.getStoredEndpoint() !== null;
  }

  private getStoredEndpoint(): string | null {
    if (typeof window === 'undefined') {
      return null;
    }
    return localStorage.getItem(STORAGE_KEY_SNS_ENDPOINT);
  }

  private initPermissionWatcher(): void {
    if (typeof window === 'undefined') {
      return;
    }

    // 1. Re-check permissions when returning to the tab / window
    window.addEventListener('focus', () => {
      this.syncPermissionStatus();
    });

    // 2. Browser Permissions API live change event
    if ('permissions' in navigator) {
      navigator.permissions
        .query({ name: 'notifications' as PermissionName })
        .then((status) => {
          status.onchange = () => {
            this.syncPermissionStatus();
          };
        })
        .catch(() => {});
    }
  }

  syncPermissionStatus(): void {
    if (typeof window === 'undefined' || !('Notification' in window)) {
      return;
    }

    const currentPerm = Notification.permission;
    this.permission.set(currentPerm);

    if (currentPerm !== 'granted') {
      const storedArn = this.getStoredEndpoint();
      if (storedArn) {
        localStorage.removeItem(STORAGE_KEY_SNS_ENDPOINT);
        this.snsEndpointArn.set(null);
        this.unregisterEndpointOnBackend(storedArn);
      }
      this.isSnsSubscribed.set(false);
      return;
    }

    const hasStored = this.getStoredEndpoint() !== null;
    this.isSnsSubscribed.set(hasStored);
  }

  private unregisterEndpointOnBackend(endpointArn: string): void {
    firstValueFrom(
      this.http.delete(`${this.apiBase}/v1/notifications/sns-endpoint`, {
        params: { endpoint_arn: endpointArn },
      })
    ).catch((err) => {
      console.warn('Backend endpoint unregister warning:', err);
    });
  }

  async requestPermission(): Promise<boolean> {
    if (!this.isSupported()) {
      return false;
    }

    try {
      const result = await Notification.requestPermission();
      this.permission.set(result);
      this.syncPermissionStatus();
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
          if (!navigator.serviceWorker.controller) {
            await navigator.serviceWorker.register('/sw.js');
          }
          const reg = await navigator.serviceWorker.ready;
          let sub = await reg.pushManager.getSubscription();

          if (!sub && environment.vapidPublicKey) {
            sub = await reg.pushManager.subscribe({
              userVisibleOnly: true,
              applicationServerKey: urlBase64ToUint8Array(environment.vapidPublicKey),
            });
          }

          if (sub) {
            const endpointParts = sub.endpoint.split('/');
            pushToken = endpointParts[endpointParts.length - 1];
          }
        } catch (subErr) {
          console.warn('Native PushManager subscription failed or VAPID key not configured:', subErr);
        }
      }

      if (!pushToken) {
        Swal.fire({
          icon: 'warning',
          title: 'Push Service Unavailable',
          text: 'A browser push subscription could not be established. Please verify that notifications and Service Workers are supported and enabled in your browser.',
          confirmButtonColor: '#09090b',
        });
        return false;
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

  async notify(title: string, body?: string, url?: string): Promise<void> {
    if (!this.isSupported()) {
      return;
    }

    if (this.permission() !== 'granted') {
      const granted = await this.requestPermission();
      if (!granted) {
        return;
      }
    }

    const options: NotificationOptions = {
      body: body || 'Security event detected',
      icon: '/favicon.svg',
      badge: '/favicon.svg',
      tag: `mg-alert-${Date.now()}`,
      data: { url: url || '/' },
    };

    if (typeof Notification !== 'undefined' && Notification.permission === 'granted') {
      try {
        new Notification(title, options);
        return;
      } catch {
        // Fall through to Service Worker showNotification if constructor fails
      }
    }

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
