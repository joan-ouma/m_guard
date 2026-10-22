import { Component, signal, AfterViewInit, ElementRef, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-about',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './about.component.html',
  styleUrls: ['./about.component.css', '../public-theme.css'],
})
export class AboutComponent implements AfterViewInit {
  private elementRef = inject(ElementRef);
  mobileMenuOpen = signal<boolean>(false);
  activeTestimonial = signal<number>(1);

  toggleMobileMenu(): void {
    this.mobileMenuOpen.update((open) => !open);
  }

  setTestimonial(index: number): void {
    this.activeTestimonial.set(index);
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
      '.story-text, .story-img, .accordion-item, .features-item, .team-item, .testimonial, .cta-wrapper'
    );
    elements.forEach((el: Element) => {
      el.classList.add('mg-scroll-anim');
      observer.observe(el);
    });
  }
}
