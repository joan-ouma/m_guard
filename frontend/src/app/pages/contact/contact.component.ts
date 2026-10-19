import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-contact',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './contact.component.html',
  styleUrl: './contact.component.css',
})
export class ContactComponent {
  mobileMenuOpen = signal<boolean>(false);
  activeLocation = signal<number>(0);

  toggleMobileMenu(): void {
    this.mobileMenuOpen.update((open) => !open);
  }

  nextLocation(): void {
    this.activeLocation.update((curr) => (curr + 1) % 3);
  }

  prevLocation(): void {
    this.activeLocation.update((curr) => (curr - 1 + 3) % 3);
  }

  setLocation(index: number): void {
    this.activeLocation.set(index);
  }
}
