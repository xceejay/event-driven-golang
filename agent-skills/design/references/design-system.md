# Lime UI Design System — Full Token Reference

Read this file before writing any code. It contains every design decision for both dashboard and landing page modes.

## Table of Contents
1. Color Tokens
2. Typography
3. Spacing & Radius
4. Shared Component Patterns
5. Dashboard-Specific Patterns
6. Landing Page-Specific Patterns
7. Animations & Micro-interactions
8. Navigation Patterns
9. Responsive Breakpoints

---

## 1. Color Tokens

Use CSS custom properties on `:root` and a `.dark` class (or React state / Svelte store / Vue reactive) to toggle themes.

### Light Mode
| Token | Value | Usage |
|---|---|---|
| --bg-page | #F5F5F0 | Page background (warm off-white, NOT pure white) |
| --bg-card | #FFFFFF with 1px border #E8E8E3 | Card surfaces, section containers |
| --bg-card-inner | #F9F9F5 | Nested elements inside cards |
| --bg-section-alt | #EDEDEA | Alternating section background (landing pages) |
| --text-primary | #1A1A1A | Headings, metric numbers, hero text |
| --text-secondary | #8A8A8A | Labels, metadata, subtitles, body text |
| --text-muted | #B0B0A8 | Placeholders, disabled text, footer links |
| --border-subtle | #E8E8E3 | Card borders, dividers, input borders |
| --accent | #C8F542 | Primary accent — lime/chartreuse |
| --accent-text | #1A1A1A | Text on accent-colored backgrounds |
| --accent-hover | #B8E232 | Accent button hover state (slightly darker) |
| --chart-inactive | #E8E8E3 | Inactive bar chart bars, ring backgrounds |
| --warning | #F5A623 | Warning/cancelled status icons |

### Dark Mode
| Token | Value | Usage |
|---|---|---|
| --bg-page | #111111 | Page background (deep charcoal) |
| --bg-card | #1A1A1A with 1px border #2A2A2A | Card surfaces |
| --bg-card-inner | #222222 | Nested elements inside cards |
| --bg-section-alt | #161616 | Alternating section background |
| --text-primary | #F5F5F0 | Headings, hero text |
| --text-secondary | #7A7A7A | Labels, body text |
| --text-muted | #4A4A4A | Placeholders, disabled, footer |
| --border-subtle | #2A2A2A | Card borders |
| --accent | #C8F542 | Same lime accent |
| --accent-text | #1A1A1A | Text on accent backgrounds |
| --accent-hover | #D4FF2B | Accent hover (slightly brighter on dark) |
| --chart-inactive | #2A2A2A | Inactive bars, ring backgrounds |
| --warning | #F5A623 | Warning icons |

### Accent Usage Rules
The lime accent (#C8F542) appears ONLY on:
- Primary CTA button fills (both dashboard and landing page)
- Active status dots and "Live" badges (dashboard)
- "NOW" pill in timelines (dashboard)
- Highlighted chart elements — active bars, ring segments, sparklines (dashboard)
- Hero CTA buttons (landing page)
- Pricing card "recommended" border highlight (landing page)
- Feature card icon accents or decorative dots (landing page)
- Hover underlines on nav links (landing page)
- Nothing else. Restraint is what makes the accent impactful.

---

## 2. Typography

Import from Google Fonts or Fontshare. Pick ONE of these — do not mix:
- DM Sans — `https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600;700&display=swap`
- Outfit — `https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700&display=swap`
- Satoshi — `https://api.fontshare.com/v2/css?f=satoshi@400,500,700&display=swap`

### Dashboard Type Scale
| Role | Size | Weight | Letter-spacing | Color |
|---|---|---|---|---|
| Big metric numbers | 36-48px | 700 | -0.02em | --text-primary |
| Card section headings | 16-18px | 600 | 0 | --text-primary |
| Body / descriptions | 14px | 400 | 0 | --text-secondary |
| Small labels / metadata | 12-13px | 400-500 | 0.01em | --text-secondary |
| Pill badge text | 12px | 500 | 0.02em | varies |

### Landing Page Type Scale
| Role | Size (desktop) | Size (mobile) | Weight | Letter-spacing |
|---|---|---|---|---|
| Hero headline | 56-72px | 36-44px | 700 | -0.03em |
| Section headline | 36-44px | 28-32px | 700 | -0.02em |
| Section subheadline | 18-20px | 16-18px | 400 | 0 |
| Feature card title | 20-24px | 18-20px | 600 | -0.01em |
| Body text | 16-18px | 15-16px | 400 | 0 |
| Nav links | 14-15px | 14px | 500 | 0.01em |
| Footer text | 13-14px | 13px | 400 | 0 |
| Stats / big numbers | 44-56px | 32-40px | 700 | -0.02em |
| Pricing amount | 48-56px | 36-44px | 700 | -0.02em |
| Testimonial quote | 18-20px | 16-18px | 400 italic | 0 |

### Metric Layout Pattern (Dashboard)
Large number first, label below in small muted text:
```
$824,592          84%            1,738
Company Balance   Active         active trips
```

### Hero Text Pattern (Landing Page)
Big headline first, subtext below, CTAs below that:
```
Build something
extraordinary.

Ship faster with tools that actually work.
No complexity, no compromises.

[ Get Started ]   [ See Demo → ]
```

---

## 3. Spacing & Radius

### Shared
| Token | Value |
|---|---|
| Card border-radius | 16-20px |
| Inner element radius | 10-12px |
| Button radius | 9999px (pill shape) |
| Pill badge radius | 9999px |
| Button padding | 12px 28px |

### Dashboard-Specific
| Token | Value |
|---|---|
| Card padding | 24-32px |
| Card gap (grid) | 16-20px |
| Sidebar width | 64-72px |
| Top bar height | 56-64px |

### Landing Page-Specific
| Token | Value |
|---|---|
| Content max-width | 1200px |
| Content horizontal padding | 24px (mobile) / 40px (tablet) / 64px (desktop) |
| Section vertical padding | 80-120px |
| Feature card padding | 32-40px |
| Pricing card padding | 36-48px |
| Nav height | 64-72px |
| Nav horizontal padding | 24-64px (responsive) |
| Footer vertical padding | 48-64px |

---

## 4. Shared Component Patterns

### Pill Badges
- Rounded-full (border-radius: 9999px), padding: 4px 12px
- Light mode: #F0F0EB background, --text-secondary text
- Active/accent state: --accent background, --accent-text text
- Small icon or emoji before text when relevant

### Buttons
- **Primary**: background --accent, color --accent-text, rounded-full, font-weight 600, hover: --accent-hover
- **Secondary**: transparent background, 1px solid --border-subtle, color --text-primary, rounded-full, hover: --bg-card-inner
- **Ghost** (landing pages): no border, color --text-secondary, hover: color --text-primary, with trailing arrow →

### Cards
```css
.card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: 18px;
  padding: 28px;
  transition: transform 0.2s ease;
}
.card:hover { transform: scale(1.005); }
```
No drop shadows. Separation comes from border + background contrast.

### Theme Toggle
Small pill-shaped toggle in the nav bar with sun/moon icons. Toggles a `.dark` class on `<html>` or uses React state.

---

## 5. Dashboard-Specific Patterns

### Card Anatomy
Every dashboard card follows three parts:
1. **Header Row** — flexbox space-between: title (left) + pill/filter/menu (right)
2. **Main Content** — metric number, chart, feed, map, or status widget
3. **Footer** — optional labels, legends, or action buttons

### Grid Layout
```css
.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 18px;
}
.card-wide { grid-column: span 2; }
```

### Bar Chart (SVG or div-based)
- 7 bars for M T W T F S S
- Rounded top (border-radius: 6px 6px 0 0), width ~32-40px
- Default fill: --chart-inactive
- Highlighted bar: --accent fill, slightly taller, tooltip bubble above
- Animate upward on mount with staggered delays (50ms per bar)

### Donut / Ring Charts (SVG)
```svg
<svg viewBox="0 0 120 120">
  <circle cx="60" cy="60" r="50" fill="none" stroke="var(--chart-inactive)" stroke-width="12" stroke-linecap="round"/>
  <circle cx="60" cy="60" r="50" fill="none" stroke="var(--accent)" stroke-width="12" stroke-linecap="round"
    stroke-dasharray="314" stroke-dashoffset="50" transform="rotate(-90 60 60)"/>
  <text x="60" y="60" text-anchor="middle" dominant-baseline="central"
    font-size="24" font-weight="700" fill="var(--text-primary)">84%</text>
</svg>
```
- stroke-dasharray = 2πr ≈ 314 for r=50
- stroke-dashoffset = 314 × (1 - percentage/100)
- Animate from dashoffset=314 to target on mount

### Sparklines (SVG)
- Thin path (stroke-width: 2), stroke: --accent, fill: none
- Optional area fill: --accent at 8-10% opacity
- No axes or grid — just the flowing line
- viewBox: "0 0 120 40", smooth cubic beziers

### Activity / Feed List
```
┌─────────────────────────────────────────┐
│ ◉  Trip #88219                      ›   │
│    Completed · 2 mins ago               │
│    Driver: Sarah J. · $42.50            │
└─────────────────────────────────────────┘
```
- Left: 40px circular icon container
- Center: title (semibold 14px) + metadata (13px, --text-secondary)
- Right: chevron arrow in --text-muted

### Progress Timeline
```
  ●    ●    ●   [NOW]   ○    ○
 6AM  12PM  6PM         12AM
```
- Past: --text-muted filled circles, current: "NOW" accent pill, future: empty circles

### Status Card with Avatar
- 48-56px circular avatar, green online dot (--accent) at bottom-right
- Name semibold 16px, metadata in --text-secondary 13px

### Map Placeholder
- Faint SVG lines suggesting roads, 1-2 accent pin markers, faint place labels

---

## 6. Landing Page-Specific Patterns

### Navigation Bar
```css
.landing-nav {
  position: sticky;
  top: 0;
  z-index: 50;
  height: 68px;
  background: var(--bg-page);
  border-bottom: 1px solid var(--border-subtle);
  backdrop-filter: blur(12px);
  background: color-mix(in srgb, var(--bg-page) 85%, transparent);
}
```
- Left: logo (text or icon, font-weight 700, 20px)
- Center: nav links (14-15px, --text-secondary, hover: --text-primary with accent underline)
- Right: theme toggle + primary CTA button (accent filled, small size)
- Mobile: hamburger icon → slide-out or dropdown menu

### Hero Section
The hero is the most important section. It sets the tone for the entire page.
```css
.hero {
  padding: 120px 0 100px;
  text-align: center; /* or left-aligned with visual on right */
}
```
**Layout options** (pick the one that fits the product):
- **Centered**: headline + subtext + CTAs centered, optional product screenshot/mockup below
- **Split**: headline + subtext + CTAs on left, product visual or illustration on right (60/40 split)

**Headline**: 56-72px, bold, --text-primary, tight letter-spacing. Should be punchy — 3-8 words max on the main line. Can use a second line. Optionally highlight one word or phrase in --accent.

**Subtext**: 18-20px, --text-secondary, max-width 600px, 1-3 sentences. Explains the value prop without jargon.

**CTA row**: primary (accent filled) + secondary (outlined or ghost with →), side by side, centered or left-aligned to match layout.

**Optional elements below CTAs**:
- Small trust line: "Trusted by 2,000+ teams" with mini logos
- Product screenshot in a soft-rounded card frame with subtle shadow
- Abstract decorative element (gradient mesh, dots grid, or accent-colored shapes)

### Logo Bar / Social Proof Strip
A horizontal row of partner/customer logos in --text-muted opacity:
```css
.logo-bar {
  padding: 40px 0;
  border-top: 1px solid var(--border-subtle);
  border-bottom: 1px solid var(--border-subtle);
}
.logo-bar img {
  height: 24-32px;
  opacity: 0.4;
  filter: grayscale(100%);
  transition: opacity 0.2s;
}
.logo-bar img:hover { opacity: 0.8; }
```
- Use flexbox with `justify-content: center` and `gap: 48px`
- 4-6 logos, all desaturated and muted until hovered
- Above or below: small label "Trusted by leading teams" in 13px --text-muted

### Feature Grid
3-column grid of feature cards (2-column on tablet, stacked on mobile):
```css
.feature-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
}
```

**Feature Card Anatomy**:
1. **Icon or visual** (top): 48px accent-tinted icon, or a small SVG illustration, or an emoji in a 56px rounded-square container with --bg-card-inner background
2. **Title** (20-24px, semibold, --text-primary)
3. **Description** (15-16px, --text-secondary, 2-3 sentences max)

Feature cards use the same `.card` styling as dashboard cards — soft border, rounded corners, hover scale.

**Alternate layout**: for 3-6 features, a 2-column layout with larger cards can work:
```
┌─────────────────────────┐  ┌─────────────────────────┐
│ 🔒                      │  │ ⚡                      │
│ Enterprise Security     │  │ Lightning Fast           │
│ Description text here   │  │ Description text here    │
│ that wraps to 2 lines.  │  │ that wraps to 2 lines.  │
└─────────────────────────┘  └─────────────────────────┘
```

### Stats / Metrics Banner
A horizontal strip showing 3-4 big numbers — reuses the dashboard metric pattern but in a full-width section:
```
        2,400+              99.9%              <50ms             150+
      Active users       Uptime SLA      Response time      Integrations
```
- Numbers: 44-56px bold, --text-primary
- Labels: 14px, --text-secondary
- Section background: --bg-section-alt for contrast
- Optional: numbers animate (count up) when scrolled into view

### Pricing Section
3-column grid of pricing cards. The recommended plan is visually elevated:

**Standard pricing card**:
- Border: 1px solid --border-subtle
- Padding: 36-48px
- Plan name: 16px semibold
- Price: 48-56px bold, --text-primary, with period ("/mo") in 16px --text-secondary
- Feature list: checkmarks (accent color) + feature text, 14-15px, comfortable line-height
- CTA button: secondary (outlined) at bottom

**Recommended pricing card**:
- Border: 2px solid --accent (lime highlight)
- "Recommended" or "Popular" pill badge at top in --accent
- CTA button: primary (accent filled) instead of secondary
- Optional: very subtle accent glow or background tint

```css
.pricing-card.recommended {
  border: 2px solid var(--accent);
  position: relative;
}
.pricing-card.recommended::before {
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: inherit;
  background: var(--accent);
  opacity: 0.03;
  z-index: -1;
}
```

### Testimonials
Two layout options:

**Card grid** (2-3 testimonials):
```
┌────────────────────────┐  ┌────────────────────────┐
│ "Quote text here that  │  │ "Another quote text    │
│  wraps nicely."        │  │  that wraps nicely."   │
│                        │  │                        │
│ 👤 Name                │  │ 👤 Name                │
│    Role, Company       │  │    Role, Company       │
└────────────────────────┘  └────────────────────────┘
```

**Single spotlight** (1 big testimonial):
- Large quote text (20px, italic, --text-primary), centered
- Decorative oversized quotation mark in --accent at 15% opacity behind the text
- Avatar (48px circle) + name + role below, centered

### Final CTA Section
A full-width section that drives conversion. Sits just above the footer:
```css
.final-cta {
  padding: 80px 0;
  text-align: center;
  background: var(--bg-section-alt);
  border-radius: 24px; /* if contained within max-width */
}
```
- Headline: 36-44px bold, "Ready to get started?" or similar
- Subtext: 18px, --text-secondary, 1 sentence
- Primary CTA button (large size: 16px text, 16px 36px padding)
- Optional: secondary link below ("No credit card required" or "Talk to sales →")

### Footer
```css
.footer {
  padding: 56px 0 32px;
  border-top: 1px solid var(--border-subtle);
}
```
- Multi-column link grid (4 columns: Product, Company, Resources, Legal)
- Column header: 14px semibold --text-primary
- Links: 14px --text-muted, hover: --text-secondary
- Bottom row: copyright text (13px --text-muted) + social icons (--text-muted, hover: --text-primary)
- Responsive: 2 columns on tablet, stacked on mobile

### Email Capture / Waitlist Input
For waitlist or newsletter signup:
```
┌──────────────────────────────────┬──────────────┐
│  Enter your email                │  Join Waitlist │
└──────────────────────────────────┴──────────────┘
```
- Input: --bg-card-inner background, 1px --border-subtle border, rounded-full, 16px padding
- Button: attached to input (or below on mobile), accent filled, rounded-full
- Input + button can be wrapped in a single rounded-full container for a sleek look
- Helper text below: 13px --text-muted ("We'll never spam you")

---

## 7. Animations & Micro-interactions

### Shared
```css
.card:hover { transform: scale(1.005); }
.btn { transition: background-color 0.15s ease, transform 0.1s ease; }
.btn:active { transform: scale(0.98); }
```

### Dashboard-Specific
```css
/* Live badge pulse */
@keyframes pulse-glow {
  0%, 100% { box-shadow: 0 0 0 0 rgba(200, 245, 66, 0.4); }
  50% { box-shadow: 0 0 0 8px rgba(200, 245, 66, 0); }
}
.badge-live { animation: pulse-glow 2s ease-in-out infinite; }

/* Bar chart mount */
.bar { transition: height 0.6s cubic-bezier(0.34, 1.56, 0.64, 1); }

/* Ring chart mount */
.ring-active { transition: stroke-dashoffset 1s cubic-bezier(0.4, 0, 0.2, 1); }
```

### Landing Page-Specific

**Scroll-triggered fade-in** — sections and cards fade up into view as the user scrolls:
```css
.fade-in-up {
  opacity: 0;
  transform: translateY(24px);
  transition: opacity 0.6s ease, transform 0.6s ease;
}
.fade-in-up.visible {
  opacity: 1;
  transform: translateY(0);
}
```
Use IntersectionObserver in JS to add `.visible` when elements enter viewport. Stagger child elements with `transition-delay` (each card 100ms later).

**Number count-up** — stats animate from 0 to target when scrolled into view:
```javascript
// Trigger when .stats-section enters viewport
function countUp(el, target, duration = 1500) {
  let start = 0;
  const step = (timestamp) => {
    if (!start) start = timestamp;
    const progress = Math.min((timestamp - start) / duration, 1);
    el.textContent = Math.floor(progress * target).toLocaleString();
    if (progress < 1) requestAnimationFrame(step);
    else el.textContent = target.toLocaleString() + (el.dataset.suffix || '');
  };
  requestAnimationFrame(step);
}
```

**Nav background transition** — nav starts transparent, gains background on scroll:
```css
.landing-nav { transition: background 0.3s ease, border-color 0.3s ease; }
.landing-nav.scrolled {
  background: color-mix(in srgb, var(--bg-page) 95%, transparent);
  border-bottom-color: var(--border-subtle);
}
```

**Hover effects on feature cards**:
```css
.feature-card:hover {
  transform: translateY(-4px);
  border-color: var(--accent);
  transition: transform 0.2s ease, border-color 0.2s ease;
}
```

---

## 8. Navigation Patterns

### Dashboard Navigation
**Sidebar** (left, icon-only):
- Width: 64-72px, full viewport height
- Icons: 24px, centered, 48px hit target
- Active: --accent color or accent pill behind icon
- Inactive: --text-muted
- Top: app logo. Bottom: notification bell, settings gear

**Top Bar**:
- Height: 56-64px, transparent background
- Logo + nav tab pills + search icon + user avatar with role badge

### Landing Page Navigation
**Sticky horizontal nav**:
- Height: 64-72px
- Logo (left) + links (center) + CTA button (right)
- Blur backdrop on scroll
- Mobile: hamburger → dropdown or slide-out panel with full nav links + CTA

---

## 9. Responsive Breakpoints

### Dashboard
```css
@media (min-width: 1024px) { .dashboard-grid { grid-template-columns: repeat(3, 1fr); } .sidebar { display: flex; } }
@media (min-width: 768px) and (max-width: 1023px) { .dashboard-grid { grid-template-columns: repeat(2, 1fr); } .sidebar { display: none; } }
@media (max-width: 767px) { .dashboard-grid { grid-template-columns: 1fr; } .sidebar { display: none; } }
```

### Landing Page
```css
@media (min-width: 1024px) {
  .feature-grid, .pricing-grid { grid-template-columns: repeat(3, 1fr); }
  .hero { padding: 120px 0 100px; }
  .hero h1 { font-size: 64px; }
}
@media (min-width: 768px) and (max-width: 1023px) {
  .feature-grid, .pricing-grid { grid-template-columns: repeat(2, 1fr); }
  .hero h1 { font-size: 48px; }
  .section { padding: 64px 0; }
}
@media (max-width: 767px) {
  .feature-grid, .pricing-grid { grid-template-columns: 1fr; }
  .hero { padding: 80px 0 60px; text-align: center; }
  .hero h1 { font-size: 36px; }
  .section { padding: 48px 0; }
  .logo-bar { flex-wrap: wrap; gap: 24px; }
  .footer-grid { grid-template-columns: repeat(2, 1fr); }
  .stats-row { flex-direction: column; gap: 32px; }
}
```
