import { PublicFooterComponent } from '../../shared/public-footer/public-footer.component';
import { Component, signal, AfterViewInit, ElementRef, ViewChild, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-about',
  standalone: true,
  imports: [CommonModule, RouterLink, PublicFooterComponent],
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

  @ViewChild('track') track?: ElementRef<HTMLElement>;

  scrollToTestimonial(index: number): void {
    const container = this.track?.nativeElement;
    const card = container?.children[index - 1] as HTMLElement | undefined;
    if (!container || !card) return;
    container.scrollTo({ left: card.offsetLeft - container.offsetLeft, behavior: 'smooth' });
    this.activeTestimonial.set(index);
  }

  onTestimonialScroll(): void {
    const container = this.track?.nativeElement;
    if (!container) return;
    let closest = 0;
    let best = Number.POSITIVE_INFINITY;
    Array.from(container.children).forEach((child, i) => {
      const offset = Math.abs((child as HTMLElement).offsetLeft - container.offsetLeft - container.scrollLeft);
      if (offset < best) {
        best = offset;
        closest = i;
      }
    });
    this.activeTestimonial.set(closest + 1);
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
