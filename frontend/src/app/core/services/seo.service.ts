import { Injectable, inject } from '@angular/core';
import { Title, Meta } from '@angular/platform-browser';
import { Router, NavigationEnd, ActivatedRoute } from '@angular/router';
import { DOCUMENT } from '@angular/common';
import { filter } from 'rxjs/operators';

export interface SeoData {
  title?: string;
  description?: string;
  keywords?: string;
  ogImage?: string;
  ogType?: string;
  noIndex?: boolean;
}

const DEFAULT_SEO: Required<SeoData> = {
  title: 'MGuard | Smart Motorcycle GPS Tracking & Security',
  description:
    'Real-time motorcycle tracking, intelligent geofencing, remote security lock, and automated theft prevention built for boda boda riders.',
  keywords:
    'motorcycle tracking, GPS bike security, boda boda tracking, Kenya bike tracking, emergency alerts, live route history, MGuard',
  ogImage: '/motorbike.png',
  ogType: 'website',
  noIndex: false,
};

@Injectable({
  providedIn: 'root',
})
export class SeoService {
  private readonly titleService = inject(Title);
  private readonly metaService = inject(Meta);
  private readonly router = inject(Router);
  private readonly activatedRoute = inject(ActivatedRoute);
  private readonly document = inject(DOCUMENT);

  private canonicalLinkElement: HTMLLinkElement | null = null;

  init(): void {
    // Listen to route changes and update SEO metadata automatically
    this.router.events
      .pipe(filter((event) => event instanceof NavigationEnd))
      .subscribe(() => {
        let route = this.activatedRoute;
        while (route.firstChild) {
          route = route.firstChild;
        }

        const data: SeoData = route.snapshot.data || {};
        const title = route.snapshot.title || data.title;

        this.updateTags({
          ...data,
          title: title || DEFAULT_SEO.title,
        });
      });
  }

  updateTags(config: SeoData): void {
    const title = config.title ? `${config.title}` : DEFAULT_SEO.title;
    const description = config.description || DEFAULT_SEO.description;
    const keywords = config.keywords || DEFAULT_SEO.keywords;
    const ogImage = config.ogImage || DEFAULT_SEO.ogImage;
    const ogType = config.ogType || DEFAULT_SEO.ogType;
    const currentUrl = typeof window !== 'undefined' ? window.location.href : '';

    // Document title
    this.titleService.setTitle(title);

    // Standard Meta Tags
    this.setOrUpdateMeta('name', 'description', description);
    this.setOrUpdateMeta('name', 'keywords', keywords);
    this.setOrUpdateMeta(
      'name',
      'robots',
      config.noIndex ? 'noindex, nofollow' : 'index, follow, max-snippet:-1, max-image-preview:large, max-video-preview:-1'
    );

    // Open Graph / Facebook / WhatsApp
    this.setOrUpdateMeta('property', 'og:site_name', 'MGuard');
    this.setOrUpdateMeta('property', 'og:title', title);
    this.setOrUpdateMeta('property', 'og:description', description);
    this.setOrUpdateMeta('property', 'og:type', ogType);
    this.setOrUpdateMeta('property', 'og:image', ogImage);
    if (currentUrl) {
      this.setOrUpdateMeta('property', 'og:url', currentUrl);
    }

    // Twitter Card Tags
    this.setOrUpdateMeta('name', 'twitter:card', 'summary_large_image');
    this.setOrUpdateMeta('name', 'twitter:site', '@MGuard');
    this.setOrUpdateMeta('name', 'twitter:title', title);
    this.setOrUpdateMeta('name', 'twitter:description', description);
    this.setOrUpdateMeta('name', 'twitter:image', ogImage);

    // Canonical Link
    this.updateCanonicalUrl(currentUrl);
  }

  private setOrUpdateMeta(attribute: 'name' | 'property', key: string, content: string): void {
    if (this.metaService.getTag(`${attribute}="${key}"`)) {
      this.metaService.updateTag({ [attribute]: key, content });
    } else {
      this.metaService.addTag({ [attribute]: key, content });
    }
  }

  private updateCanonicalUrl(url: string): void {
    if (!this.document) return;

    if (!this.canonicalLinkElement) {
      this.canonicalLinkElement = this.document.querySelector('link[rel="canonical"]');
      if (!this.canonicalLinkElement) {
        this.canonicalLinkElement = this.document.createElement('link');
        this.canonicalLinkElement.setAttribute('rel', 'canonical');
        this.document.head.appendChild(this.canonicalLinkElement);
      }
    }

    if (url) {
      this.canonicalLinkElement.setAttribute('href', url.split('?')[0]);
    }
  }
}
