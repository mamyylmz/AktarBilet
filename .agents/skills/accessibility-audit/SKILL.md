---
name: accessibility-audit
description: Audit web interface for WCAG 2.2 AA compliance, semantic HTML, keyboard-only navigation, ARIA labeling, responsive layouts, and accessible text alternatives.
---

# Accessibility Audit Skill

## 1. Purpose
The `accessibility-audit` skill ensures that AktarBilet's web frontend ([web/src/](file:///home/dilunsel/projects/AktarBilet/web/src/)) delivers an accessible, robust, and responsive user experience compliant with WCAG 2.2 Level AA guidelines. It combines automated automated DOM scans (`axe-core`) with manual keyboard navigation checks, responsive viewport reflow verification, and accessible text alternatives for graphical transit elements.

## 2. When to Use
- Reviewing UI components, modals, tables, and form inputs for screen-reader and keyboard accessibility.
- Auditing mobile viewport responsiveness (320px to 1440px) and touch targets.
- Verifying color contrast ratios across both Light and Dark themes.
- Auditing transit maps and heatmap visualizations for accessible textual fallbacks.

## 3. When NOT to Use
- Backend API performance profiling or database migration audits.
- Adding complex, distracting decorative animations or experimental UI frameworks.
- Testing non-browser endpoints or CLI tools.

## 4. Inputs and Existing Tools
- **Frontend Codebase:** [web/src/App.jsx](file:///home/dilunsel/projects/AktarBilet/web/src/App.jsx), [web/src/components/](file:///home/dilunsel/projects/AktarBilet/web/src/components/), [web/src/features/](file:///home/dilunsel/projects/AktarBilet/web/src/features/).
- **Automated Verification:**
  - `npm --prefix web run build`
  - `npm --prefix web run lint`
  - Playwright test runner: `npx playwright test qa/tests/accessibility.spec.js` (or axe runner).
- **Existing Reports:** [AGY_ACCESSIBILITY_PERFORMANCE.md](file:///home/dilunsel/projects/AktarBilet/qa/reports/AGY_ACCESSIBILITY_PERFORMANCE.md), [AGY_UX_ACCESSIBILITY_FINAL.md](file:///home/dilunsel/projects/AktarBilet/qa/reports/archive/AGY_UX_ACCESSIBILITY_FINAL.md).

## 5. Procedure
1. **Semantic HTML & Labeling Audit:**
   - Every input field (`<input>`, `<select>`, `<textarea>`) must have an associated `<label>` connected via matching `id` and `htmlFor` attributes, or an explicit `aria-label` / `aria-labelledby`.
   - Modals and dialogs must declare `role="dialog"`, `aria-modal="true"`, and connect an accessible title via `aria-labelledby`.
   - Tables must include `<thead>`, `<th>` with `scope="col"`, and structured `<tbody>` rows (see [Table.jsx](file:///home/dilunsel/projects/AktarBilet/web/src/components/ui/Table.jsx)).
2. **Keyboard Navigation & Focus Management:**
   - Verify that all interactive elements (buttons, links, form inputs, role switchers) can be navigated sequentially using only the `Tab` and `Shift+Tab` keys.
   - All interactive controls must display a distinct, high-contrast visible focus outline (e.g., `focus-visible:ring-2 focus-visible:ring-primary`).
   - When a modal dialog opens, keyboard focus must move inside the modal, trap focus while open, and return to the trigger element when closed via `Escape` or the close button.
3. **Contrast & Theme Verification:**
   - Verify text-to-background contrast ratio is at least **4.5:1** for normal text and **3:1** for large text across both **Light** and **Dark** theme modes.
   - Status indicators (e.g., success, danger, warning badges) must not rely on color alone; text labels or semantic icons must accompany color cues.
4. **Responsive Reflow & Touch Targets:**
   - Test layout at 320px, 375px (mobile), 768px (tablet), and 1280px (desktop) without horizontal scrolling (except dedicated data tables).
   - Ensure touch targets on mobile viewports are at least **44 × 44 CSS pixels**.
   - Verify mobile hamburger menu opens, closes cleanly, and toggles `aria-expanded` state.
5. **Accessible Text Alternatives for Maps:**
   - Graphical transit maps and heatmaps must provide an accessible, tabular or textual alternative listing stations, bus lines, and crowding levels for screen-reader users.

## 6. Invariants
- **No Reliance on Automated Scans Alone:** A 0-violation axe-core scan is necessary but not sufficient. Manual keyboard traversal and reflow verification are required.
- **No Hidden Traps:** Users must never be trapped in a dropdown, modal, or input widget without an obvious keyboard escape path (`Escape` key).
- **Graceful State Handling:** Every asynchronous data view must provide explicit Loading, Empty, and Error states with appropriate `role="status"` or `role="alert"`.

## 7. Expected Evidence
- Axe-core scan results showing 0 critical or serious violations.
- Recorded keyboard traversal log confirming logical tab order.
- Responsive rendering verification at 320px width showing zero overlapping text or clipping.

## 8. Stop Conditions
- Form inputs without accessible labels trigger immediate audit failure.
- Inaccessible modal dialogs that allow focus to escape to background content trigger failure.
- Critical contrast failures (<3:1 for text) block release approval.

## 9. Agent Ownership
- **Codex CLI (Independent QA):** Executes automated axe-core scans, performs keyboard traversal verification, and files accessibility bug reports.
- **Antigravity (Development Owner):** Implements accessible React components, semantic HTML structures, ARIA attributes, and CSS focus styles in frontend code.

