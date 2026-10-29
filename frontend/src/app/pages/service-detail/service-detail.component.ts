import { PublicFooterComponent } from '../../shared/public-footer/public-footer.component';
import { Component, computed, effect, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { map } from 'rxjs';

interface ServiceDetail {
  title: string;
  intro: string;
  image: string;
  imageAlt: string;
  points: string[];
}

const SERVICES: Record<string, ServiceDetail> = {
  'live-gps': {
    title: 'Live GPS Tracking',
    intro: 'Real-time GPS positioning updates every few seconds. Monitor live location, speed, and full travel routes directly from your smartphone.',
    image: '/assets/landing/svc-live-gps.jpg',
    imageAlt: 'Phone showing live GPS location on a map',
    points: [
      'Location updates every few seconds',
      'Live location and speed on your phone',
      'Full travel routes kept in your history',
    ],
  },
  'engine-cut-off': {
    title: 'Remote Engine Cut-Off',
    intro: 'Instantly turn off your motorcycle engine remotely from your phone with a single tap. Disables the bike immediately in case of theft or hijacking.',
    image: '/assets/landing/svc-engine-cutoff.jpg',
    imageAlt: 'Motorcycle ignition key',
    points: [
      'One tap from your phone or the web dashboard',
      'The bike changes state only after the device confirms the command',
      'Use it at once if the bike is taken',
    ],
  },
  'motion-alerts': {
    title: 'Touch & Movement Alerts',
    intro: 'Advanced motion sensors detect unexpected movement, towing, or tilt while parked. Triggers immediate phone notifications so you can act instantly.',
    image: '/assets/landing/svc-motion-alert.jpg',
    imageAlt: 'Red alert light signalling a security event',
    points: [
      'Detects movement, towing and tilt while parked',
      'Instant notifications on your phone',
      'Stays active whenever the bike is parked',
    ],
  },
  'safe-zones': {
    title: 'Safe Operating Zones',
    intro: 'Define safe operating areas around your home, boda stage, or delivery routes. Receive instant alerts on your phone if your motorcycle crosses outside.',
    image: '/assets/landing/svc-safe-zones.jpg',
    imageAlt: 'Map showing a marked operating area',
    points: [
      'Set home, stage and delivery areas',
      'Get an alert when the bike leaves a zone',
      'Zones are set up from your dashboard',
    ],
  },
  'trip-history': {
    title: 'Daily Trip & Route History',
    intro: 'Review your full travel routes, customer trips, and daily distance traveled. Keep accurate, verifiable records of your riding activity on your phone.',
    image: '/assets/landing/svc-trip-history.jpg',
    imageAlt: 'Phone mounted on a motorcycle handlebar showing a route',
    points: [
      'The full route for every trip',
      'Daily distance travelled',
      'Records kept in your account',
    ],
  },
  'police-proof': {
    title: 'Police & Recovery Proof',
    intro: 'Every location and movement record is securely timestamped. Export clean, verified incident logs and live coordinates for police tracking and rapid vehicle recovery.',
    image: '/assets/landing/svc-recovery-proof.jpg',
    imageAlt: 'Laptop showing recovery and incident charts',
    points: [
      'Timestamped location and movement records',
      'Export clean incident logs',
      'Share live coordinates with police',
    ],
  },
};

@Component({
  selector: 'app-service-detail',
  standalone: true,
  imports: [RouterLink, PublicFooterComponent],
  templateUrl: './service-detail.component.html',
  styleUrls: ['./service-detail.component.css', '../services/services.component.css', '../public-theme.css'],
})
export class ServiceDetailComponent {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private title = inject(Title);

  mobileMenuOpen = signal<boolean>(false);

  private slug = toSignal(
    this.route.paramMap.pipe(map((params) => params.get('slug') ?? '')),
    { initialValue: '' }
  );

  readonly service = computed<ServiceDetail | null>(() => SERVICES[this.slug()] ?? null);

  constructor() {
    effect(() => {
      const service = this.service();
      if (!service) {
        this.router.navigateByUrl('/services');
        return;
      }
      this.title.setTitle(`${service.title} | MGuard`);
    });
  }

  toggleMobileMenu(): void {
    this.mobileMenuOpen.update((open) => !open);
  }
}
