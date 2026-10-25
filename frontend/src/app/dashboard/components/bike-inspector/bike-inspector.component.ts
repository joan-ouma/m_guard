import { Component, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FleetBike } from '../../models/dashboard.models';

@Component({
  selector: 'app-bike-inspector',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './bike-inspector.component.html',
})
export class BikeInspectorComponent {
  readonly bikeList = input.required<FleetBike[]>();
  readonly selectedBike = input<FleetBike | null>(null);
  readonly currentTime = input<number>(Date.now());
  readonly copiedCoordinates = input<boolean>(false);
  readonly mobileSheetExpanded = input<boolean>(false);

  readonly toggleImmobilize = output<FleetBike>();
  readonly recenterMap = output<void>();
  readonly copyCoordinates = output<void>();
  readonly openRegisterModal = output<void>();
  readonly toggleMobileSheet = output<void>();
  readonly dismiss = output<void>();

  formatLastSeen(dateStr?: string | null): string {
    if (!dateStr) return 'Never reported';
    const timestamp = new Date(dateStr).getTime();
    if (isNaN(timestamp)) return 'Never reported';

    const diffSec = Math.max(0, Math.floor((this.currentTime() - timestamp) / 1000));
    if (diffSec < 5) return 'Just now';
    if (diffSec < 60) return `${diffSec}s ago`;
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) return `${diffMin}m ago`;
    const diffHours = Math.floor(diffMin / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${Math.floor(diffHours / 24)}d ago`;
  }

  isOffline(bike?: FleetBike | null): boolean {
    if (!bike) return true;
    if (!bike.lastSeenAt) return true;
    const timestamp = new Date(bike.lastSeenAt).getTime();
    if (isNaN(timestamp)) return true;
    const diffSec = (this.currentTime() - timestamp) / 1000;
    // When a device was last seen 5 min ago (300s), it's offline!
    if (diffSec > 300) return true;
    // Poor or unknown signal (CSQ 0 or 99) only drops the signal bars; the device is still reporting.
    return false;
  }

  getSignalBars(bike?: FleetBike | null): number {
    if (!bike || this.isOffline(bike)) return 0;
    const csq = bike.signalQuality ?? 0;
    if (csq >= 26) return 5;
    if (csq >= 20) return 4;
    if (csq >= 14) return 3;
    if (csq >= 8) return 2;
    if (csq >= 1) return 1;
    return 0;
  }

  getSignalLabel(bike?: FleetBike | null): string {
    if (!bike || this.isOffline(bike)) return 'Offline';
    const bars = this.getSignalBars(bike);
    if (bars === 5) return 'Full Signal (5/5)';
    if (bars === 4) return 'Strong (4/5)';
    if (bars === 3) return 'Moderate (3/5)';
    if (bars === 2) return 'Fair (2/5)';
    if (bars === 1) return 'Weak (1/5)';
    return 'No Signal';
  }
}
