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
      let clientID = localStorage.getItem('motorguard_client_device_id');
      if (!clientID) {
        clientID = 'web_' + Math.random().toString(36).substring(2, 15) + Date.now().toString(36);
        localStorage.setItem('motorguard_client_device_id', clientID);
      }
      const pushToken = clientID;

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
}
