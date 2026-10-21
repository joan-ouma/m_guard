import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';

@Component({
  selector: 'app-auth-layout',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './auth-layout.component.html',
})
export class AuthLayoutComponent {
  @Input({ required: true }) heading = '';
  @Input({ required: true }) subHeading = '';
}
