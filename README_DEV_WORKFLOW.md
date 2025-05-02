# Developer Workflow & Best Practices

## 1. Running the Dev Server
- Start your Go backend and static asset server as described in the main README.
- Ensure you have hot reload enabled (see below).

## 2. Formatting & Linting
- Use Prettier for HTML, CSS, and JS formatting. Run:
  ```
npx prettier --write "web/templates/**/*.html" "web/static/js/**/*.js" "web/static/css/**/*.css"
  ```
- Configure your editor to auto-format on save using `.prettierrc`.

## 3. Accessibility Testing
- Use Chrome DevTools Lighthouse to audit accessibility before release:
  - Open DevTools > Lighthouse > Generate report (Accessibility)
- Optionally, use [axe DevTools](https://www.deque.com/axe/devtools/) for deeper a11y checks.

## 4. Tailwind CSS Purge Verification
- After building for production, check `/static/css/output.css`.
- If the file is large (>100KB), purge may not be working. Ensure `tailwind.config.js` points to all relevant files.

## 5. Live Reload / Hot Reload
- Use a live reload tool for rapid UI iteration. Recommended options:
  - [Browsersync](https://browsersync.io/) (works with Go static server)
  - [Air](https://github.com/cosmtrek/air) (for Go backend hot reload)
- If not set up, manually refresh browser after changes.

## 6. Release Checklist
- Lint and format all files
- Run accessibility audit
- Verify Tailwind CSS purge
- Test on mobile and desktop

---

_Keep this file updated as your workflow evolves._
