import { Component, signal, AfterViewInit, ElementRef, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-contact',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './contact.component.html',
  styleUrls: ['./contact.component.css', '../public-theme.css'],
})
export class ContactComponent implements AfterViewInit {
  private elementRef = inject(ElementRef);
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

  ngAfterViewInit(): void {
    if (typeof IntersectionObserver === 'undefined') return;
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('mg-in-view');
          }
        });
      },
      { threshold: 0.1, rootMargin: '0px 0px -40px 0px' }
    );

    const elements = this.elementRef.nativeElement.querySelectorAll(
      '.contact-card, .contact-item, .location-item, .career-customer-item, .w-form'
    );
    elements.forEach((el: Element) => {
      el.classList.add('mg-scroll-anim');
      observer.observe(el);
    });
  }
}
