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
  readonly selectedBike = input.required<FleetBike>();
  readonly currentTime = input<number>(Date.now());
  readonly copiedCoordinates = input<boolean>(false);
  readonly mobileSheetExpanded = input<boolean>(false);

  readonly toggleImmobilize = output<FleetBike>();
  readonly recenterMap = output<void>();
  readonly copyCoordinates = output<void>();
  readonly openRegisterModal = output<void>();
  readonly toggleMobileSheet = output<void>();

  formatLastSeen(dateStr?: string | null): string {
    if (!dateStr) return '3s ago';
    const timestamp = new Date(dateStr).getTime();
    if (isNaN(timestamp)) return '3s ago';

    const diffSec = Math.max(0, Math.floor((this.currentTime() - timestamp) / 1000));
    if (diffSec < 5) return '3s ago';
    if (diffSec < 60) return `${diffSec}s ago`;
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) return `${diffMin}m ago`;
    const diffHours = Math.floor(diffMin / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${Math.floor(diffHours / 24)}d ago`;
  }
}
