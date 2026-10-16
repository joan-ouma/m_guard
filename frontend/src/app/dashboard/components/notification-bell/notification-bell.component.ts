import { Component, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NotificationItem } from '../../models/dashboard.models';
import { NotificationService } from '../../../core/services/notification.service';

@Component({
  selector: 'app-notification-bell',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './notification-bell.component.html',
})
export class NotificationBellComponent {
  readonly notificationService = inject(NotificationService);

  readonly notifications = input<NotificationItem[]>([]);
  readonly unreadCount = input<number>(0);

  readonly viewNotification = output<NotificationItem>();
  readonly markAllRead = output<void>();

  readonly isOpen = signal<boolean>(false);

  togglePopover(): void {
    this.isOpen.update((v) => !v);
  }

  onView(item: NotificationItem): void {
    this.isOpen.set(false);
    this.viewNotification.emit(item);
  }

  onMarkAllRead(): void {
    this.markAllRead.emit();
  }

  async onEnableSns(): Promise<void> {
    await this.notificationService.registerWithSNS();
  }

  async onDisableSns(): Promise<void> {
    await this.notificationService.unregisterFromSNS();
  }
}
