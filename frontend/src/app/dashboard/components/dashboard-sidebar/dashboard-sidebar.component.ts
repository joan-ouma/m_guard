import { Component, inject, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { PwaService } from '../../../core/services/pwa.service';

@Component({
  selector: 'app-dashboard-sidebar',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterLinkActive],
  templateUrl: './dashboard-sidebar.component.html',
})
export class DashboardSidebarComponent {
  readonly pwa = inject(PwaService);

  readonly mobileDrawerOpen = input<boolean>(false);
  readonly user = input<any>();

  readonly closeMobileDrawer = output<void>();
  readonly openGeofenceModal = output<void>();
  readonly logout = output<void>();

  getUserInitials(): string {
    const raw = (this.user()?.name || this.user()?.email || '').trim();
    if (!raw) return 'U';
    const parts = raw.split(/\s+/).filter(Boolean);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return raw.slice(0, 2).toUpperCase();
  }
}
