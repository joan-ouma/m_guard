import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FleetBike, NotificationItem } from '../../models/dashboard.models';
import { NotificationBellComponent } from '../notification-bell/notification-bell.component';

@Component({
  selector: 'app-dashboard-header',
  standalone: true,
  imports: [CommonModule, NotificationBellComponent],
  templateUrl: './dashboard-header.component.html',
})
export class DashboardHeaderComponent {
  readonly selectedBike = input<FleetBike | null>(null);
  readonly hasBikes = input.required<boolean>();
  readonly isBreachActive = input<boolean>(false);
  readonly alertBannerText = input<string>('');
  readonly notifications = input<NotificationItem[]>([]);
  readonly unreadCount = input<number>(0);

  readonly toggleMobileDrawer = output<void>();
  readonly simulateSuddenStop = output<void>();
  readonly openGeofenceModal = output<void>();
  readonly viewNotification = output<NotificationItem>();
  readonly markAllRead = output<void>();
}
