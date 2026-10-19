import { Component, inject, signal, OnInit } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { Router } from '@angular/router';

@Component({
  selector: 'app-legal',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './legal.component.html',
  styleUrl: './legal.component.css',
})
export class LegalComponent implements OnInit {
  private router = inject(Router);
  private location = inject(Location);

  activeTab = signal<'terms' | 'privacy'>('terms');
  lastUpdated = 'October 14, 2026';

  ngOnInit(): void {
    if (this.router.url.includes('privacy')) {
      this.activeTab.set('privacy');
    } else {
      this.activeTab.set('terms');
    }
  }

  setTab(tab: 'terms' | 'privacy'): void {
    this.activeTab.set(tab);
    window.scrollTo({ top: 0, behavior: 'smooth' });
    this.router.navigate([`/${tab}`], { replaceUrl: true });
  }

  goBack(): void {
    if (window.history.length > 1) {
      this.location.back();
    } else {
      this.router.navigate(['/login']);
    }
  }
}
