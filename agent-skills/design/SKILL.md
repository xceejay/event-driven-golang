---
name: lime-ui
description: Build polished web interfaces with a distinctive neumorphic-adjacent aesthetic — warm neutrals, lime-green accent, soft rounded surfaces, geometric sans-serif type, light/dark mode, subtle animations. TWO modes — (1) Dashboards/admin panels with card grids, metric widgets, SVG charts, activity feeds, and (2) Landing pages with hero sections, feature grids, pricing tables, testimonials, logo bars, CTAs. Use whenever the user asks to build a dashboard, admin panel, analytics view, landing page, marketing page, product page, SaaS homepage, waitlist page, or any modern UI. Also trigger for "lime design", "lime dashboard", "lime landing page", or dark/light card-based layouts. Even for a plain "build me a landing page" or "make a dashboard" — this skill produces far better results than a generic approach.
---

# Lime UI Skill

A design system for building production-grade web interfaces with a distinctive aesthetic: warm neutrals, a single lime-green accent, soft neumorphic-adjacent surfaces, and subtle animations. Works across any framework — React, Svelte, Vue, vanilla HTML/CSS/JS, or whatever the user specifies.

## Two modes

This skill covers two interface types that share the same design DNA but use different layout patterns:

**Dashboard Mode** — for operational dashboards, admin panels, analytics views, monitoring consoles. Uses a sidebar + card grid layout with data visualization widgets.

**Landing Page Mode** — for marketing pages, product homepages, SaaS landing pages, waitlist pages, portfolio sites. Uses a full-width scrolling layout with hero sections, feature grids, social proof, and CTAs.

Determine which mode to use based on the user's request. If ambiguous, ask. Some projects may combine both (e.g., a SaaS site with a public landing page AND an authenticated dashboard) — handle each page in its respective mode.

## When you receive a request

1. **Read the design reference** at `references/design-system.md` in this skill's directory. It contains the full token spec for colors, typography, spacing, component patterns, and animations for BOTH modes. Do not skip this step.
2. **Determine the mode** — dashboard or landing page (or both).
3. **Identify the domain and framework** — the user will describe their content and may specify a stack. Default to React JSX + Tailwind if unspecified. Adapt to any framework requested.
4. **Plan the layout** — for dashboards, plan the card grid and widget types. For landing pages, plan the section flow (hero → features → social proof → pricing → CTA → footer).
5. **Build the artifact** — produce a single file that implements the full interface with realistic mock content. Everything should feel alive — no placeholder text or empty states.

## Core aesthetic principles

These are non-negotiable across BOTH modes:

- **Warm neutrals, not pure white.** Light mode uses off-white (#F5F5F0), never #FFFFFF. Dark mode uses deep charcoal (#111111), not generic grays.
- **Single accent color.** Lime/chartreuse (#C8F542) used sparingly — only on primary CTAs, active indicators, highlights, and key visual moments. The accent pops because it's rare.
- **Soft surfaces.** Cards and sections feel pressed into the page (neumorphic-adjacent), not floating above it. No harsh Material Design shadows. Faint borders + background contrast.
- **Large rounded corners.** 16-20px on cards and sections, 12px on inner elements, full-round on buttons and pills.
- **Geometric sans-serif typography.** Use "DM Sans", "Outfit", "General Sans", or "Satoshi" — never Inter, Roboto, Arial, or system fonts.
- **Subtle animations.** Scroll-triggered fade-ins for landing pages, mount transitions for dashboard charts, gentle hover effects everywhere. Nothing flashy or distracting.
- **Light/dark mode.** Always implement a theme toggle. Both modes share the same components — only surface and text colors change.

## Dashboard mode layout

```
┌──────┬──────────────────────────────────────────┐
│ Icon │  Logo  │ Nav Tabs │ Search │ User Avatar  │
│ Nav  ├──────────────────────────────────────────┤
│ Bar  │  ┌─────────┐ ┌─────────┐ ┌─────────┐    │
│      │  │ Card 1  │ │ Card 2  │ │ Card 3  │    │
│      │  └─────────┘ └─────────┘ └─────────┘    │
│      │  ┌─────────┐ ┌─────────┐ ┌─────────┐    │
│      │  │ Card 4  │ │ Card 5  │ │ Card 6  │    │
│      │  └─────────┘ └─────────┘ └─────────┘    │
│      │  ┌─────────┐ ┌─────────┐ ┌─────────┐    │
│      │  │ Card 7  │ │ Card 8  │ │ Card 9  │    │
│      │  └─────────┘ └─────────┘ └─────────┘    │
└──────┴──────────────────────────────────────────┘
```

- Left sidebar: narrow icon-only vertical nav
- Top bar: logo, navigation pills, search, user avatar
- Main area: responsive CSS grid (3 cols → 2 → 1)
- Widget types: metric cards, bar charts, ring/donut charts, sparklines, activity feeds, progress timelines, status cards, map placeholders

## Landing page mode layout

```
┌──────────────────────────────────────────────────┐
│  Logo          Nav Links          CTA Button      │  ← Sticky nav
├──────────────────────────────────────────────────┤
│                                                    │
│          Hero: Big headline + subtext              │
│          Primary CTA    Secondary CTA              │
│          Optional hero visual / mockup             │
│                                                    │
├──────────────────────────────────────────────────┤
│        Logo bar / social proof strip               │
├──────────────────────────────────────────────────┤
│                                                    │
│    ┌────────┐  ┌────────┐  ┌────────┐             │
│    │Feature │  │Feature │  │Feature │             │
│    │  Card  │  │  Card  │  │  Card  │             │
│    └────────┘  └────────┘  └────────┘             │
│                                                    │
├──────────────────────────────────────────────────┤
│        Metrics / stats banner                      │
├──────────────────────────────────────────────────┤
│                                                    │
│    ┌────────┐  ┌────────┐  ┌────────┐             │
│    │Pricing │  │Pricing │  │Pricing │             │
│    │  Card  │  │  Card  │  │  Card  │             │
│    └────────┘  └────────┘  └────────┘             │
│                                                    │
├──────────────────────────────────────────────────┤
│        Testimonials / quotes                       │
├──────────────────────────────────────────────────┤
│        Final CTA section                           │
├──────────────────────────────────────────────────┤
│        Footer with links + copyright               │
└──────────────────────────────────────────────────┘
```

- Full-width sections, content constrained to max-width (1200px) and centered
- Generous vertical spacing between sections (80-120px)
- Alternating section backgrounds for visual rhythm
- No sidebar — navigation is a sticky horizontal top bar

## What to avoid

These are hallmarks of generic AI output. Do not produce any of them:

- Pure white (#FFFFFF) backgrounds
- Generic shadows (`box-shadow: 0 2px 8px rgba(0,0,0,0.1)`)
- Inter, Roboto, Arial, or system-ui fonts
- Blue accents or purple gradients
- Recharts/Chart.js for dashboard charts (hand-craft SVG instead)
- Material Design elevation patterns
- Harsh borders or dividers — use spacing and subtle tone shifts
- Stock-photo placeholder images — use abstract shapes, gradients, or SVG illustrations
- Generic hero patterns with centered text over a gradient blob

## Responsive behavior

**Dashboard:**
- Desktop (≥1024px): 3-column grid with sidebar
- Tablet (768-1023px): 2-column grid, sidebar collapses
- Mobile (<768px): single column stack

**Landing Page:**
- Desktop (≥1024px): full layout, 3-column feature/pricing grids
- Tablet (768-1023px): 2-column grids, adjusted spacing
- Mobile (<768px): single column, stacked sections, hamburger nav

## After building

Populate everything with realistic mock content appropriate to the user's domain. No "Lorem ipsum", no empty states, no "Your Company" placeholders. The output should look like a live production site on first render.
