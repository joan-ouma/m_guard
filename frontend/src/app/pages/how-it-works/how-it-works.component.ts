import { PublicFooterComponent } from '../../shared/public-footer/public-footer.component';
import { Component, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-how-it-works',
  standalone: true,
  imports: [CommonModule, RouterLink, PublicFooterComponent],
  templateUrl: './how-it-works.component.html',
  styleUrls: ['./how-it-works.component.css', '../contact/contact.component.css', '../public-theme.css'],
})
export class HowItWorksComponent {
  mobileMenuOpen = signal<boolean>(false);

  toggleMobileMenu(): void {
    this.mobileMenuOpen.update((open) => !open);
  }
}
