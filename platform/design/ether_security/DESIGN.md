# Design System Specification: The Private Curator

## 1. Overview & Creative North Star
The "Private Curator" is a design system built on the philosophy of **Fortified Serenity**. It rejects the cluttered, noisy patterns of traditional productivity software in favor of an editorial, high-end experience that feels both impenetrable and effortless. 

Our Creative North Star is **"Digital Sanctuary."** This means every pixel must communicate security through precision and calm through negative space. We move beyond "standard" UI by employing intentional asymmetry, tonal layering instead of borders, and a sophisticated interplay between the tech-forward `Primary` purples and the gallery-clean `Surface` tones. The layout should feel like a premium physical dossier—structured, tactile, and deeply private.

---

## 2. Colors & Atmospheric Tonalities
This system utilizes a sophisticated palette of deep amethyst and cool slates to establish authority without aggression.

### The "No-Line" Rule
**Explicit Instruction:** 1px solid borders are strictly prohibited for sectioning or containment. Boundaries must be defined through background color shifts or subtle tonal transitions.
*   *Correct:* A `surface_container_low` card sitting on a `surface` background.
*   *Incorrect:* A white box with a gray `#CCCCCC` border.

### Surface Hierarchy & Nesting
Treat the UI as a series of physical layers. Use the following tiers to define importance:
*   **Base Layer:** `surface` (#f9f9fc) for the main application canvas.
*   **Sectional Layer:** `surface_container_low` (#f3f3f6) for secondary sidebars or grouping content.
*   **Interactive Layer:** `surface_container_lowest` (#ffffff) for primary cards or data entry areas. This "pops" against the darker base.
*   **The "Glass & Gradient" Rule:** For floating elements (modals, dropdowns), use semi-transparent `surface` colors with a 20px `backdrop-blur`. Apply a subtle linear gradient (from `primary` to `primary_container`) on hero CTAs to provide a "liquid silk" texture that flat hex codes cannot replicate.

---

## 3. Typography: The Editorial Voice
We utilize a dual-font strategy to balance high-end editorial aesthetics with technical readability.

*   **The Voice (Display & Headlines):** `Manrope`. We use Manrope for all `display` and `headline` levels. Its geometric yet warm curves suggest a modern, bespoke architecture. 
    *   *Rule:* Use `display-lg` (3.5rem) with tight letter-spacing (-0.02em) for landing moments to create an authoritative, "magazine-style" hierarchy.
*   **The Engine (Title, Body, Labels):** `Inter`. Chosen for its legendary legibility in data-dense environments.
    *   *Rule:* All `body-md` text should utilize a line-height of 1.5 to ensure maximum breathing room, reinforcing the "Privacy" aspect—nothing feels cramped or hidden.

---

## 4. Elevation & Depth: Tonal Layering
Traditional drop shadows are often "dirty." In this system, we use **Ambient Light Simulation**.

*   **The Layering Principle:** Depth is achieved by "stacking." A `surface_container_high` element should only ever sit on a `surface_container` or lower.
*   **Ambient Shadows:** If an element must float (e.g., a floating action button), use an extra-diffused shadow: `box-shadow: 0 12px 40px rgba(27, 0, 99, 0.06);`. Note the tint: the shadow is a low-opacity version of `on_primary_fixed`, not black.
*   **The "Ghost Border":** For high-stakes accessibility (e.g., input focus), use the `outline_variant` token at 20% opacity. Never 100%.
*   **Glassmorphism:** Use `surface_container_lowest` at 80% opacity with a blur to create a "Frosted Amethyst" effect for navigation bars.

---

## 5. Components

### Buttons: The Tactile Interaction
*   **Primary:** Background `primary` (#5427e6), Text `on_primary`. Shape: `md` (0.75rem). No shadow on rest; a soft `primary_container` glow on hover.
*   **Secondary:** Background `secondary_container`, Text `on_secondary_container`. Use this for "Safe" actions.
*   **Tertiary:** Ghost style. No container. Text `primary`. Use for low-priority discovery.

### Cards & Lists: The Separation of Logic
*   **Rule:** Forbid divider lines.
*   **Implementation:** Separate list items by increasing the vertical `surface_container` padding or by alternating subtle background shifts (e.g., Item 1: `surface`, Item 2: `surface_container_low`).
*   **Roundedness:** All cards must use `lg` (1rem) corners to soften the "industrial" feel of secure software.

### Input Fields: The Secure Entry
*   **Default:** `surface_container_highest` background, no border.
*   **Focus State:** A soft 2px outer glow using `primary_fixed` and a transition of the background to `surface_container_lowest`.
*   **Privacy Feature:** Password/Sensitive fields should use a "masked blur" effect when not in focus to reinforce the privacy theme visually.

### The "Shield" Chip
A bespoke component for this system. A selection chip using `secondary_fixed` background with a small icon prefix. Used to show "Encrypted" or "Verified" status.

---

## 6. Do's and Don'ts

### Do:
*   **Embrace Asymmetry:** Align headings to the left while keeping data summaries to the right with generous "white-space-gutters" (32px or 48px).
*   **Use Tonal Depth:** Layer a `white` card on a `surface_container_low` background. It feels premium and intentional.
*   **Prioritize Typography:** Let the size of `Manrope` headlines do the work that buttons and lines usually do.

### Don't:
*   **Don't use 100% Black:** For text, always use `on_surface` (#1a1c1e). It is softer on the eyes and feels more sophisticated.
*   **Don't use "Default" Shadows:** Avoid the standard `0 2px 4px rgba(0,0,0,0.5)`. It looks cheap.
*   **Don't Cram:** If a screen feels full, increase the page height. Privacy requires room to breathe; "crowded" feels "unsecure."
*   **No Hard Borders:** If you feel the urge to draw a line, use a background color change instead.