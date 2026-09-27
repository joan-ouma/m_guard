import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-auth-layout',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './auth-layout.component.html',
})
export class AuthLayoutComponent {
  @Input({ required: true }) heading = '';
  @Input({ required: true }) subHeading = '';
}
