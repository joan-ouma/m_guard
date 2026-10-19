import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-about',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './about.component.html',
  styleUrl: './about.component.css',
})
export class AboutComponent {
  mobileMenuOpen = signal<boolean>(false);
  activeAccordion = signal<number>(1);
  activeTestimonial = signal<number>(1);

  toggleMobileMenu(): void {
    this.mobileMenuOpen.update((open) => !open);
  }

  setAccordion(index: number): void {
    this.activeAccordion.set(index);
  }

  setTestimonial(index: number): void {
    this.activeTestimonial.set(index);
  }
}
