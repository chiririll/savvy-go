# 02 — Collapse the mail settings while mail is off

## Problem

Administration → System always shows the whole SMTP form (host, protocol,
port, user, password, from, skip-verify, timeout) plus the "Save" button, even
on a server that will never send mail. It is nine rows of noise above the
other settings, and `MailForm` still shows them when `mail.enabled` is false.

## Where

- `resources/ts/components/features/mail/MailSection.tsx`: `MailForm` renders
  the status line, then one `<form>` with the `enabled` `ToggleRow` and every
  field, then (only if `mail.enabled`) the test-email form.
- `MailStatus.enabled` means "configured, valid and loaded"; the toggle in the
  form is `draft.enabled`. They differ: a user can tick the toggle before
  saving a valid config.
- Existing test: `MailSection.test.tsx` (case with `enabled: false` and a
  `problem`).

## Fix

1. Keep the status line and the `enabled` toggle always visible.
2. Render the other fields (host … timeout, save button) only while
   `draft.enabled` is true, inside a `Collapsible` (`components/ui/collapsible`
   if present, else a plain conditional) so the toggle opens them.
3. When the saved config has a `problem` (configured but ignored), keep the
   fields open so the admin can fix it.
4. Switching the toggle off and saving must still send the full `MailInput`
   with `enabled: false` (the backend takes every setting on `PUT /admin/mail`),
   keeping the stored password (`password` omitted).
5. Read-only config (`writable === false`): fields stay visible but disabled
   only when the server has mail configured.

## Tests (write first)

- Mail off, no problem: only status and toggle are in the DOM.
- Toggle on: fields appear; off: they go away and the draft values survive a
  second toggle.
- Mail off with a `problem`: fields are visible.
- Saving with the toggle off sends `enabled: false` and no `password`.

## Done when

A server without mail shows two lines in this section; nothing is lost when
the admin toggles it.
