You are a Staff Product Designer and Design Systems Architect with 18 years of experience crafting interfaces for high-traffic retail, hospitality, and embedded systems. You have led design at Sonos, Square, and Aesop Digital. You are tasked with implementing the complete UI layer for **Astra-Service**, a self-checkout kiosk system running on 9:16 vertical touchscreens (1080x1920 and 1440x2560). The aesthetic direction is **"Living Weave"** — a biophilic, wabi-sabi inspired design language adapted for calm retail efficiency. This is not a mood board. This is the exact design specification that engineering will implement pixel-for-pixel.

**System Context & Constraints**
- Display: 9:16 vertical, 1080x1920 (primary) and 1440x2560 (premium lanes). Touch-only, no keyboard, no mouse.
- Viewing distance: 18–28 inches. User may be standing, holding items, or with a child.
- Lighting: Variable retail lighting from dim boutiques to bright fluorescent. The UI must remain legible in 1000+ lux environments.
- Performance: 60fps animations. First paint < 800ms. Route transition < 200ms.
- Stack: React 19 (StrictMode), TypeScript 5.5 (strictest config), Tailwind CSS 4, Framer Motion, XState v5, Zustand, TanStack Query.
- Accessibility: WCAG 2.2 AA minimum, touch targets ≥ 56px, full TalkBack/VoiceOver support, high contrast mode toggle.

**Design Philosophy: "Calm Commerce"**
Adapt the "Living Weave" biophilic philosophy to retail self-service. The interface should feel like a premium boutique experience — not a frantic fast-food terminal, not a sterile medical device. The emotional goal is **confident tranquility**: the user feels the system is sophisticated, unhurried, yet completely competent. Every transaction should feel intentional, every touch rewarded with subtle physicality.

- **Imperfection is controlled**: Organic blob shapes and soft textures exist in the background, but interactive elements are geometrically precise for usability.
- **Material honesty**: The UI suggests linen, aged paper, and moss, but through performance-optimized CSS — no heavy image assets, no WebGL, no canvas backgrounds.
- **Breathable density**: Retail kiosks often panic and cram information. Astra-Service uses generous spacing, progressive disclosure, and tiered information architecture. Primary action is always obvious; secondary actions are discoverable but not hidden.
- **Speed through serenity**: Fast animations (150–250ms) with ease-out curves. No bouncy, playful spring physics. The motion is fluid, like turning a heavy page in a quality notebook.

---

### 1. Unified Color System (Retail-Adapted)

The palette must work in bright retail environments while maintaining the "Living Weave" soul. Colors are slightly more saturated than the original craft spec to ensure legibility under fluorescent lighting, but remain muted and sophisticated.

**Base & Neutrals**:
- `Linen`: `#F5F3EF` — primary background for all screens. Never pure white.
- `Warm Cream`: `#FEF7E0` — secondary background, cart summary bands, receipt paper simulation.
- `Card Surface`: `#FFFFFF` at 88% opacity over Linen — content cards. Use `backdrop-blur-[8px]` only when layered over imagery.
- `Charcoal`: `#2D2A26` — primary text. Darker than the original for retail contrast.
- `Stone`: `#6B6862` — secondary text, prices, metadata, disabled states.
- `Taupe`: `#C4B8A8` — dividers, hairlines, inactive track backgrounds.
- `Clay`: `#B8A99A` — subtle borders, stitched line effects.

**Biophilic Accents**:
- `Moss`: `#5A7A5C` — primary action, progress indicators, success states, active navigation. Slightly more saturated than original for visibility.
- `Amber`: `#B87E6B` — CTAs, call-to-action buttons, total price highlights, "Pay" actions. Warm, inviting, urgent without aggression.
- `Denim`: `#4A5D70` — informational elements, help buttons, secondary actions, link text.
- `Deep Forest`: `#1A3A2A` — high-emphasis text, dark mode primary text (see dark mode section).
- `Pale Mint`: `#E8F5E9` — tinted backgrounds for success states, offline mode banner, P2P sync active indicator.
- `Soft Rose`: `#C4A4A4` — error states, voided items, payment failure. Muted, not alarming red.

**Functional Colors**:
- `Offline`: `#D4A843` — warm amber for offline mode banner. Suggests candlelight, not danger.
- `Sync Active`: `#5A7A5C` with subtle pulse — P2P mesh active.
- `Printer`: `#6B6862` — thermal printer status, paper low warnings.

**Opacity Rules**:
- Background textures: 2–5% opacity. Must be invisible on mobile screenshots but present in person.
- Glassmorphic overlays: `bg-white/75` with `backdrop-blur-[10px]` maximum. Never frosted glass over busy backgrounds.
- Shadows: `shadow-[0_4px_24px_rgba(45,42,38,0.08)]` for cards. `shadow-[0_8px_32px_rgba(45,42,38,0.12)]` for modals. Diffused, never sharp.
- Disabled states: 40% opacity + `grayscale-[0.5]`, never just gray text.

---

### 2. Typography System (Distance-Optimized)

**Font Stack**:
- **Display / Headings**: `Cormorant Garamond`, weight 600. Used for brand voice moments: "Welcome," "Thank you," "Payment Confirmed." `letter-spacing: -0.02em`. Line height: 1.1.
- **UI / Body**: `Inter`, weight 400–500. All functional text, prices, buttons, labels. `letter-spacing: -0.01em`. Line height: 1.5.
- **Monospace / Data**: `IBM Plex Mono`, weight 400. Used for transaction IDs, timestamps, P2P node IDs, offline queue counts. `letter-spacing: 0em`. Line height: 1.4.
- **Fallback stack**: `system-ui, -apple-system, sans-serif` for Inter; `Georgia, serif` for Cormorant.

**Scale (9:16 Optimized)**:
- **Hero / Attract Title**: 56px (1080px width) / 72px (1440px width). Cormorant. Centered.
- **Screen Title**: 36px. Inter 500. Left-aligned or centered depending on screen.
- **Section Header**: 24px. Inter 500. Used for cart categories, payment methods.
- **Body**: 18px. Inter 400. Minimum 16px for accessibility — never smaller.
- **Label / Tag**: 13px. Inter 500. Uppercase, `tracking-[0.08em]`, Stone color. Used for "Item Count," "Subtotal," "Tax."
- **Price**: 28px. Inter 600. Charcoal. Tabular nums (`font-variant-numeric: tabular-nums`) so totals align.
- **Total Price**: 42px. Inter 600. Amber color. Tabular nums.
- **Micro / Caption**: 14px. Inter 400. Stone. Used for modifiers, "tap to edit," legal text.

**Text Rendering**:
- `-webkit-font-smoothing: antialiased` on all text.
- `text-wrap: pretty` for headings, `balance` for short paragraphs.
- Prices and quantities must use `font-variant-numeric: tabular-nums` to prevent jitter during animations.

---

### 3. Layout & Spatial System (9:16 Grid)

**Viewport Strategy**:
- Design in a 9:16 frame: 375px wide × 812px tall logical (1080×1920 physical at 3x), or 430px × 932px logical (1440×2560 at ~3.3x).
- Safe areas: 24px horizontal padding minimum. 32px top padding (status bar zone). 40px bottom padding (thumb zone / home indicator).
- Touch heatmap: Primary actions live in the bottom 25% of screen (thumb zone). Secondary actions in top 15%. Content scrolls in the middle 60%.

**Grid**:
- 4-column grid with 16px gutters. Columns are fluid within padding.
- Base unit: 8px. All spacing, padding, margins, border-radius must be multiples of 8px.

**Z-Index Architecture**:
- `z-0`: Background texture layer
- `z-10`: Base content, scrollable lists
- `z-20`: Sticky headers, floating cart summary
- `z-30`: Modals, bottom sheets, dialogs
- `z-40`: Toasts, notifications, offline banner
- `z-50`: System overlays, emergency cancel, attract loop screensaver

**Status Bar (Persistent Top Zone, 48px height)**:
- Left: P2P sync status icon (subtle dot: Moss = synced, Amber = syncing, Stone = offline). Tap reveals mesh detail.
- Center: Time (Inter 14px, Stone).
- Right: Network/cloud status (icon only, no text). Printer status if connected.
- Background: Transparent over content. If content scrolls underneath, a `bg-linen/80 backdrop-blur-[8px]` fades in after 100px scroll.

**Bottom Action Bar (Persistent, 96px height)**:
- Background: `bg-linen` with a top border of `1px solid Taupe`.
- Left: "Cancel Order" (text button, Denim, only visible when cart has items).
- Center/Right: Primary action button ("Review Cart →", "Pay $24.50", "Start New Order").
- On payment screens: This bar transforms to show the Verifone terminal animation area.

---

### 4. Textures & Materiality (Performance-First)

**Background Stack**:
1. **Base**: `Linen` `#F5F3EF` solid fill.
2. **Weave Texture**: CSS `repeating-linear-gradient` at 2% opacity, 4px intervals, both axes. Must be a single CSS class, not an image.
3. **Paper Grain**: Static SVG noise overlay, `baseFrequency="0.8"`, 3% opacity, `mix-blend-mode: multiply`. Disable on hardware with < 4GB RAM via `@media (prefers-reduced-data: reduce)` or runtime detection.

**Card Surfaces**:
- Background: `bg-white/85` with no blur unless over imagery.
- **Stitched Border**: `::after` pseudo-element with `inset: 5px`, `border: 1px dashed rgba(61,58,54,0.12)`, `border-radius: inherit`, `pointer-events: none`.
- Shadow: `shadow-[0_2px_12px_rgba(45,42,38,0.06)]`.
- On active/tap: `scale-[0.98]` with 100ms transition. No lift on mobile — prevents parallax motion sickness.

**Hero / Attract Screen**:
- Full-screen `Linen` background.
- 2–3 organic blob shapes in `Moss` at 4% opacity and `Amber` at 3% opacity. Slow morph animation (12s cycle, `will-change: transform`).
- Center: Brand mark (Astra-Service wordmark in Cormorant) + "Tap to start" in Inter 18px, Stone, with a gentle pulse opacity.
- No buttons visible until tap. The entire screen is the tap target.

---

### 5. Component Specifications (Kiosk-Native)

**Primary Button (CTA)**:
- Height: 64px (minimum 56px touch target + visual padding).
- Background: `bg-amber` (`#B87E6B`). Text: `text-white`, Inter 500, 18px.
- Border-radius: `rounded-full` (pill shape). Full-width on mobile, max-w-md centered on wider kiosks.
- Shadow: `shadow-[0_4px_16px_rgba(184,126,107,0.3)]`.
- Hover (touch-start): `brightness-110 scale-[1.01]`.
- Active (touch-end): `scale-[0.98] translate-y-[1px]`.
- Disabled: `opacity-50 grayscale` with "Loading..." spinner (Moss color, 20px).
- Focus: `focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-moss`.

**Secondary Button**:
- Height: 56px.
- Background: `bg-white/70 border border-taupe`.
- Text: Charcoal, Inter 500, 16px.
- Border-radius: `rounded-[16px]`.

**Menu Item Card**:
- Layout: Horizontal card. 96px × 96px image left (square, `rounded-[12px]`), content right.
- Image: Object-cover, with a subtle `bg-stone/10` placeholder blurhash.
- Title: Inter 500, 18px, Charcoal, 1 line max with `text-overflow: ellipsis`.
- Description: Inter 400, 14px, Stone, 2 lines max.
- Price: Inter 600, 18px, Charcoal, right-aligned.
- Modifier hint: "Customize →" in Denim, 13px, if item has modifiers.
- Tap target: Entire card. Active state: `bg-warm-cream/50` with 100ms transition.
- Stitched border on all cards.

**Quantity Stepper** (Cart Item):
- Layout: Horizontal. `-` button | `2` count | `+` button.
- Buttons: 48px × 48px circular, `bg-linen border border-taupe`. Icon: 20px, Charcoal.
- Count: Inter 600, 20px, centered, 48px min-width.
- Long-press on `+` or `-` accelerates after 500ms (repeat every 100ms).

**Cart Summary Band**:
- Position: Sticky above bottom action bar.
- Background: `bg-warm-cream/90 backdrop-blur-[8px]`.
- Content: "3 items" left (Label style), "$24.50" right (Price style).
- Expandable: Tap to expand into full cart review without leaving current screen (bottom sheet).

**Bottom Sheet**:
- Background: `bg-white/95 backdrop-blur-[12px]`.
- Border-radius: `rounded-t-[24px]`.
- Handle: 40px × 4px rounded, `bg-taupe`, centered at top for affordance.
- Entry: `translateY(100%) → translateY(0)`, 300ms, `cubic-bezier(0.16, 1, 0.3, 1)`.
- Exit: Reverse, 200ms.
- Backdrop: `bg-charcoal/20` fades in.

**Modal / Dialog**:
- Centered, max-width 90%, `rounded-[24px]`, `bg-white`, shadow deep.
- For: Item customization, payment confirmation, employee override.
- Entry: `scale(0.95) opacity(0) → scale(1) opacity(1)`, 200ms.

**Toast / Notification**:
- Position: Top center, below status bar.
- Background: `bg-charcoal text-white`.
- Border-radius: `rounded-[12px]`.
- Entry: `translateY(-20px) opacity(0) → translateY(0) opacity(1)`, 250ms.
- Auto-dismiss: 4 seconds. Progress bar strip at bottom (thin, Amber).

**Offline Banner**:
- Position: Top of screen, below status bar, full-width.
- Background: `bg-pale-mint border-b border-moss/20`.
- Text: "Working offline. Your cart is secure." Inter 14px, Moss.
- Icon: Cloud with slash, 16px.
- Dismissible after 3 seconds via auto-collapse to a small dot in status bar.

**P2P Sync Indicator**:
- Status bar dot only by default.
- Expanded (tap): Bottom sheet showing mesh topology — "Lane 1 ● Lane 2 ● Lane 3" with connecting lines. Monospace font for node IDs. Moss = healthy, Amber = syncing, Stone = partitioned.

**Payment Processing Overlay**:
- Full-screen translucent `bg-linen/90 backdrop-blur-[4px]`.
- Center: Animated organic blob (Moss, 8% opacity) with a subtle rotate.
- Text: "Processing payment..." Inter 18px, Stone.
- Below: Verifone terminal status in Monospace 13px. "Terminal: AUTHORIZING".
- Cancel: Secondary button at bottom, "Cancel" (only if terminal allows).

**Biometric Auth Prompt**:
- Modal overlay.
- Title: "Verify to complete" Cormorant 28px.
- Body: "Please use the PIN pad or present your card to the terminal."
- Visual: Animated fingerprint/card icon in Moss, subtle pulse.
- Terminal state: Live connection status to Verifone.

**Thermal Printer Feedback**:
- Small toast: "Printing receipt..." with printer icon.
- On paper low: Amber warning dot in status bar. Tap reveals "Please alert staff: paper low."

---

### 6. Screen-by-Screen Specifications

**Screen 1: Attract Loop (Idle)**
- Full-screen `Linen` with animated blobs.
- Center: "Astra" in Cormorant 56px, Charcoal. Subtitle: "Touch to begin" in Inter 18px, Stone, pulsing opacity (3s cycle).
- Bottom: Subtle scrolling text: "Self-checkout • Lane 3" in Monospace 12px, Stone.
- After 2 minutes of idle: Dim to 30% brightness (CSS filter), blobs slow to 20s cycle.
- Tap anywhere: Blobs expand outward, screen transitions to Menu with a `clipPath` circle reveal from touch point (500ms).

**Screen 2: Menu Browse**
- Top: Sticky category chips (horizontal scroll, snap-x). Chips: `bg-white/60 border border-taupe rounded-full px-4 py-2`. Active chip: `bg-moss text-white border-moss`.
- Body: Vertical scroll of Menu Item Cards. Categories are sticky headers (Inter 13px uppercase, Stone, `bg-linen/95 backdrop-blur-[4px]`).
- Right edge (optional): Floating "Cart" pill showing item count + total, appears after first item added.
- Search: Hidden by default. Pull down to reveal search bar (like iOS). Search uses debounced API calls with skeleton screens.
- Empty state: "No items found" with a small leaf illustration at 8% opacity.
- **Ghost Cart Transfer**: If a phone cart is detected via NFC/WebRTC, a bottom sheet slides up: "Cart found on your phone. Add to this kiosk?" with preview.

**Screen 3: Item Detail / Customization (Modal)**
- Entry: Bottom sheet from bottom.
- Image: Top 40% of sheet, `rounded-t-[24px]`, object-cover.
- Content: Title (Cormorant 24px), description (Inter 16px, Stone), price (Inter 28px).
- Modifiers: Radio groups or checkboxes. Each option is a row with `rounded-[12px] bg-white/50 border border-taupe` tap target. Selected state: `border-moss bg-pale-mint/30`.
- Quantity stepper.
- Primary button: "Add to cart — $8.50" full-width.
- Swipe down to dismiss (touch only).

**Screen 4: Cart Review**
- Full screen or bottom sheet (full screen if >5 items).
- Header: "Your cart" Cormorant 32px.
- List: Cart items with thumbnail, name, modifiers as subtitle, quantity stepper, line total.
- Divider: Dashed line (`border-dashed border-taupe`) between items.
- Summary: Subtotal, Tax, Total. Total in 42px Amber.
- Below total: "Tap an item to edit" in Stone 14px.
- Action bar: "← Back to menu" secondary left, "Pay $24.50 →" primary right.
- **Silent Assist**: If dwell >40s, the primary button gently pulses (opacity 0.8→1, 2s) and a subtle arrow animates toward it.

**Screen 5: Payment Auth**
- Header: "Ready to pay" Cormorant 28px.
- Cart summary: Collapsible, default collapsed (show total only).
- Payment methods: Horizontal scroll of cards. "Card / NFC", "Cash", "QR Code". Each is a large tap target (120px × 120px) with icon and label.
- Selected: `border-moss bg-pale-mint/20`.
- **Auth Trigger**: Only when "Confirm Payment" is tapped, the biometric auth modal appears. The Verifone terminal wakes up (visual connection status).
- Employee override: Hidden "Hold for 3 seconds" on corner for staff access (WebAuthn).

**Screen 6: Processing**
- Full overlay (see Component spec).
- States: "Connecting to terminal..." → "Waiting for card..." → "Authorizing..." → "Finalizing..."
- Each state change triggers a subtle haptic vibration (if supported).
- Progress: Not a bar, but a series of 4 small dots that fill (Moss) sequentially.

**Screen 7: Receipt / Confirmation**
- Background: `Warm Cream` to simulate paper.
- Success icon: Checkmark in Moss circle, drawn with SVG stroke animation (300ms).
- "Thank you" Cormorant 36px.
- Order number: Monospace 24px, Charcoal. "Order #A-7842"
- Below: "Print receipt" secondary button, "Email receipt" secondary button, "Start new order" primary button (appears after 3 seconds to prevent accidental double-tap).
- If printer fails: Toast "Printer unavailable. Receipt saved to your account."

**Screen 8: Admin / Assist (Employee Only)**
- Accessed via long-press corner + biometric auth.
- Dark theme: `bg-charcoal text-linen` (see Dark Mode).
- Monospace data tables for transactions, P2P mesh status, offline queue depth, sync logs.
- Actions: Void transaction, force sync, restart lane, open cash drawer.

---

### 7. Animation & Motion System

**Easing Definitions**:
- `ease-out-expo`: `cubic-bezier(0.16, 1, 0.3, 1)` — primary entrances.
- `ease-in-out-soft`: `cubic-bezier(0.4, 0, 0.2, 1)` — ambient motion.
- `ease-spring`: `cubic-bezier(0.34, 1.56, 0.64, 1)` — only for micro-interactions (stepper buttons).

**Timing**:
- Micro-interactions (button press, checkbox): 100–150ms.
- Layout shifts (bottom sheet, modal): 250–350ms.
- Page transitions: 300–400ms.
- Ambient (blob morph, float): 8–12s cycles.

**Specific Animations**:
- **Page Transition**: Current screen fades to 0.9 opacity and slides out left (`translateX(-5%)`). New screen slides in from right (`translateX(5%) → 0`) with fade. 300ms.
- **Cart Add**: Item thumbnail flies from menu card to floating cart pill (FLIP animation, 400ms).
- **Price Update**: Old price fades up, new price fades in from below (`translateY(4px)`). 150ms.
- **Blob Morph**: `border-radius` animates between organic shapes using CSS keyframes. `will-change: border-radius, transform`.
- **Stagger**: List items enter with 40ms stagger on first render. `opacity 0→1`, `translateY(8px)→0`.

**Reduced Motion**:
- `@media (prefers-reduced-motion: reduce)`: All animations become instant or 50ms opacity fades. Ambient blobs freeze. No fly animations.

---

### 8. Dark Mode (Retail Night / Admin)

Triggered by time-of-day (after 10 PM) or admin override.
- Background: `#1C1A17` (warm dark, not pure black).
- Card: `#2A2824` with `border-stone/10`.
- Text: `#F5F3EF` (Linen) primary, `#A8A49D` secondary.
- Accents: Moss `#7A9A7C`, Amber `#C49A8A`.
- Shadows: Reduced, use `border` instead for depth.
- Textures: Weave layer at 1% opacity only.

---

### 9. Accessibility & Inclusive Design

- **Contrast**: All text 4.5:1 minimum. Total price 7:1.
- **Touch**: 56px minimum targets. 8px spacing between adjacent targets.
- **Focus**: Visible `ring-2 ring-moss ring-offset-2` on all interactive elements. No focus trap in modals.
- **Screen Reader**: 
  - `aria-live="polite"` region for cart updates ("3 items in cart, total 24 dollars 50 cents").
  - `aria-label` on all icon buttons.
  - Route announcements via `aria-live="assertive"` on screen change.
- **Color Blindness**: Never rely on color alone. Success = checkmark + text. Error = X + text. Offline = icon + text.
- **High Contrast Mode**: `prefers-contrast: high` forces `border-charcoal` on all cards, pure black text, no transparency.
- **Cognitive**: Simple language. No jargon. "Pay" not "Initiate transaction." "Add to cart" not "Append to order."

---

### 10. Copy Tone & Voice

- **Calm, direct, respectful.** The user is in control. The system is helpful, not eager.
- **Headlines**: "Your cart," "Ready to pay," "Thank you." Not "Awesome sauce!" or "You're almost there!"
- **CTAs**: "Add to cart," "Review order," "Pay $24.50," "Start new order." Verbs first.
- **Errors**: "We couldn't connect to the payment terminal. Please try again or ask for help." Never blame the user. Never error codes without explanation.
- **Offline**: "Working offline. Everything is saved." Reassuring, not alarming.
- **Multilingual ready**: All strings in i18n keys. Default English (US). Spanish next. French third. Keys like `payment.confirmButton`, `cart.emptyState`.

---

### 11. Technical Implementation Rules

**State Management**:
- XState v5 for the global kiosk state machine (`ATTRACT`, `MENU`, `ITEM_DETAIL`, `CART`, `PAYMENT`, `PROCESSING`, `RECEIPT`, `ADMIN`).
- Zustand for ephemeral UI state (bottom sheet open, category scroll position, search query).
- TanStack Query for server state (menu API, inventory checks) with stale-while-revalidate and optimistic updates.

**Styling**:
- Tailwind CSS 4 with custom design tokens in `tailwind.config.ts`.
- All colors as CSS variables in `:root` for runtime theme switching.
- No arbitrary values in JSX. All spacing, colors, radii must be from the design token system.
- `container-type: inline-size` for container queries on kiosk size variants.

**Performance**:
- Images: AVIF with WebP fallback. Blurhash placeholders. `loading="lazy"` except above fold.
- Fonts: `font-display: swap`. Preload Cormorant and Inter.
- Bundle: Route-based code splitting. Attract screen must be < 100KB. Menu screen < 150KB.
- Animation: Only `transform` and `opacity`. No `layout`, `width`, `height`, or `top/left` animations.

**Offline Visuals**:
- When `navigator.onLine === false` or P2P mesh is active, the status bar dot changes. No blocking modals.
- If offline > 5 minutes, a subtle border appears around the screen: `border-2 border-offline/30` as ambient warning.
- Cart operations work instantly (optimistic). Sync indicator shows queue depth if > 10 items.

**P2P Mesh Visualization**:
- Admin-only bottom sheet.
- Nodes as circles, connections as lines. Animated data packets (small dots) travel along lines when sync is active.
- Uses CSS animations, not canvas, for simplicity.

**Computer Vision Overlay**:
- When produce recognition activates, a subtle scanning reticle appears over the camera preview (if shown) or a full-screen overlay with "Hold item to camera."
- Matches the color system: reticle in Moss, text in Charcoal on Linen background.

**Output Requirements**
Generate the following as production-ready code:
1. `apps/kiosk/tailwind.config.ts` — complete token system.
2. `apps/kiosk/src/styles/global.css` — base styles, textures, fonts, CSS variables.
3. `apps/kiosk/src/machines/kioskMachine.ts` — XState v5 machine with all states, guards, and actions.
4. `apps/kiosk/src/components/screens/AttractScreen.tsx` — full implementation.
5. `apps/kiosk/src/components/screens/MenuScreen.tsx` — with category chips, item list, search.
6. `apps/kiosk/src/components/screens/CartScreen.tsx` — full cart review with summary.
7. `apps/kiosk/src/components/screens/PaymentScreen.tsx` — payment method selection.
8. `apps/kiosk/src/components/screens/ProcessingScreen.tsx` — overlay with states.
9. `apps/kiosk/src/components/screens/ReceiptScreen.tsx` — confirmation.
10. `apps/kiosk/src/components/ui/` — Button, Card, BottomSheet, Toast, Stepper, StatusBar, OfflineBanner.
11. `packages/design-system/src/tokens/colors.ts` — typed color tokens.
12. `packages/design-system/src/tokens/typography.ts` — typed typography scale.
13. `packages/design-system/src/tokens/spacing.ts` — typed spacing scale.

Every file must compile. Every type must be defined. Every animation must use Framer Motion or CSS transitions with the exact easing and timing specified. Every color must reference the token system. No placeholder images — use CSS gradients or SVG patterns. No "TODO" comments. No `any` types. No unhandled promises.

Begin generating files in order. Start with the token system and global styles, then the state machine, then screens, then components. If interrupted, checkpoint exactly at the last file completed and resume without repetition.