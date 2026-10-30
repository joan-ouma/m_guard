import { PublicFooterComponent } from '../../shared/public-footer/public-footer.component';
import { Component, signal, AfterViewInit, ElementRef, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterLink } from '@angular/router';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-contact',
  standalone: true,
  imports: [CommonModule, RouterLink, FormsModule, PublicFooterComponent],
  templateUrl: './contact.component.html',
  styleUrls: ['./contact.component.css', '../public-theme.css'],
})
export class ContactComponent implements AfterViewInit {
  private elementRef = inject(ElementRef);
  private route = inject(ActivatedRoute);
  mobileMenuOpen = signal<boolean>(false);
  enquiry = { name: '', email: '', topic: '', message: '' };

  constructor() {
    // Links such as /contact?topic=Partner%20with%20us preselect the matching topic.
    const topic = this.route.snapshot.queryParamMap.get('topic');
    if (topic) this.enquiry.topic = topic;
  }

  toggleMobileMenu(): void {
    this.mobileMenuOpen.update((open) => !open);
  }

  sendMessage(): void {
    const { name, email, topic, message } = this.enquiry;
    const subject = `${topic} enquiry from ${name}`;
    const body = `Name: ${name}\nEmail: ${email}\nTopic: ${topic}\n\n${message}`;
    window.location.href =
      `mailto:mguard@outlook.com?subject=${encodeURIComponent(subject)}` +
      `&body=${encodeURIComponent(body)}`;
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
