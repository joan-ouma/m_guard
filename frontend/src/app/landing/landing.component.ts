import { PublicFooterComponent } from '../shared/public-footer/public-footer.component';
import { Component, signal, AfterViewInit, ElementRef, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-landing',
  standalone: true,
  imports: [CommonModule, RouterLink, PublicFooterComponent],
  templateUrl: './landing.component.html',
  styleUrls: ['./landing.component.css', '../pages/public-theme.css'],
})
export class LandingComponent implements AfterViewInit {
  private elementRef = inject(ElementRef);
  mobileMenuOpen = signal<boolean>(false);

  toggleMobileMenu(): void {
    this.mobileMenuOpen.update((open) => !open);
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
      '.service-card, .testimonial, .accomplishments-card, .solution-text, .help-wrap, .help-details-wrap > div, .title-wrapper'
    );
    elements.forEach((el: Element) => {
      el.classList.add('mg-scroll-anim');
      observer.observe(el);
    });
  }
}
