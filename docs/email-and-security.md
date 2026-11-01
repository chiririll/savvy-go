# Email and security

Server settings live in `config.toml`, see [Deployment](deployment.md#configuration).

## Email

Without a mail server everything works: reset and invitation links are shown for you to copy and send. With one, Go Savvy can email them. Some things need it, see [how invitations are made](spaces.md#invitations).

Set it up in **Administration → System → Email**, or in `[mailer]` in `config.toml`. While mail is off the panel shows only its status and the **Send email** switch; turn the switch on to reveal the server fields (they stay open if the saved settings have a problem). Saving in the panel writes the file and takes effect at once, no restart. **Test connection** connects and signs in with the values in the form, saved or not, without sending anything; **Save** runs the same test first and keeps the settings only if it passes (switching mail off is saved without a test). A **Send test email** button checks the saved connection and sends a message to you. If the application cannot write `config.toml` (a read-only mount), the form is locked and you edit the file and restart.

**What the emails look like.** Password links, email confirmation and invitations are sent as an HTML email with a plain-text copy for clients that show only text. Each has a button for the action and the raw link below it, a note that it can be ignored if unexpected, and the date the link is valid until (UTC, no time). Dark mode follows the reader's client. The logo is sent inside the email, so opening it makes no request to your server or anyone else's.

**Language.** English and Russian. Whoever sends a password link or an invitation picks the language of that email in the **Email language** field (it starts as their own interface language). It is used for that one email and is not saved on the account. Confirmation emails use the language of the interface of the person who asked for them. An unknown language falls back to English.

In the panel, leave the password field empty to keep the stored password; clear it on purpose to remove it. A configuration that cannot work (empty host, bad port, a `user` without a password) is refused with the field at fault, unless mail is switched off.

```toml
[mailer]
enabled = true
host = "smtp.example.com"
port = 587
protocol = "starttls"
user = "savvy@example.com"
password = "app-password"
from = "Go Savvy <savvy@example.com>"
```

| `protocol`           | Port | Connection |
|----------------------|------|------------|
| `starttls` (default) | 587  | Starts plain, upgrades to TLS. Never falls back to plain text. |
| `smtps`              | 465  | TLS from the first byte. |
| `smtp`               | 25   | No encryption. Only for a trusted relay on the same host or network. |

- The TLS certificate is checked. `skip_verify = true` accepts any, for self-signed certificates.
- `user` and `password` are optional (leave `user` empty for a relay without login). The password is plain text in `config.toml`, so make the file readable by the application only. The admin panel never shows it.
- `timeout` is seconds to wait for the server (default 10).
- A mistake in the section does not stop the server. Mail is switched off, the log says why, and the Email page shows the problem.

With mail working and `app_url` set:

- **Administration → Users → Email a reset link** sends a one-time password link (valid 1 hour). Not offered for yourself, SSO-only users, or when password sign-in is off. **Reset password** still shows a link to copy.
- **Administration → Users → New user** (no password) can email the set-password link (valid 7 days). With email confirmation on, emailing is the only option, because a link you hand over proves nothing about the mailbox. Setting the password through an emailed link also marks the email confirmed; a link you hand over never does.
- **Settings → Space** invitations by email send the link to that address only. The inviter never sees it.

## Email confirmation

An address becomes confirmed when:

- the user opens the link in a confirmation email (valid 7 days);
- they register from an invitation **email** (the link went only to that address). An account made from a copied link starts unconfirmed;
- they sign in through SSO and the provider reports the address as verified;
- a server admin uses **Mark email as confirmed** in **Administration → Users**. Changing a user's address clears the mark.

By default nothing depends on this. Turn on **Administration → System → Require email confirmation** and users with an unconfirmed address see only a "confirm your email" page (with a resend button) until they confirm. Their API tokens are blocked too. The first admin starts unconfirmed as well.

- **Safe to enable.** It can only be turned on when mail and `app_url` work and your own address is confirmed. Turning it off is always possible.
- **Restoring a backup** brings the setting back as it was. If you restore a server where it is on and all admins are unconfirmed, nobody can pass the page until mail works and someone confirms (or you restore a backup where it is off).
- **Links** are random tokens stored hashed. A link works once, a newer one replaces the older, and changing the address ends it.

**Invitation cooldown.** Any space admin can make the server email an address by inviting it. To prevent mailbox flooding, an address cannot be invited again for 1–2 hours after an invitation to it was revoked or expired, in any space. A server admin can end the wait early in **Administration → System → Invitation cooldown**, for one address or all.

## Password links

A link that sets a password works once, and a newer link for the same user replaces the older.

| Kind       | For | Valid |
|------------|-----|-------|
| Invitation | an account with no password yet (**New user** without a password) | 7 days |
| Reset      | an account with a password (**Reset password**, **Email a reset link**) | 1 hour, set `reset_ttl` in minutes |

```toml
[security]
reset_ttl = 60
```

- Using a reset link signs the user out everywhere and **deletes all their API tokens**, including connected MCP apps. They must create them again. Changing your own password in **Settings** signs out your other sessions but keeps your tokens.
- The token is part of the link's address. Pages send `Referrer-Policy: no-referrer`, so browsers do not leak it. A reverse proxy still logs full addresses by default: leave `/set-password/` and `/api/auth/password/` out of its access log, or redact them, and keep the log private.
- Passwords are at least 8 characters everywhere.

## Sign-in rate limits

These routes are throttled: sign-in, second factor, passkey sign-in, first-run registration and bootstrap password check, password reset links, invitation links and email confirmation links. Over the limit they answer `429` with a `Retry-After` header (seconds).

| Key | Default | Limit |
|-----|---------|-------|
| `ip_rate_limit` | `10` | attempts per minute from one client address, over all these routes together |
| `account_rate_limit` | `5` | sign-in attempts per minute against one email address from any address (ignoring case and surrounding spaces), right password or not |

```toml
[security]
ip_rate_limit = 10
account_rate_limit = 5
```

- `0` turns a limit off; a negative number stops the server at start.
- Counters live in memory and reset on restart.
- The client address comes from `X-Forwarded-For`. Behind a proxy that does not set it, all users share the proxy's address and the first limit. See [Behind a reverse proxy](deployment.md#behind-a-reverse-proxy).

## Two-factor lockout

- **5 tries per sign-in.** After the password step, the fifth wrong code (authenticator or recovery) ends the sign-in. The user enters the password again.
- **Lockout across sign-ins.** After **10 wrong codes in a row**, the account gets no two-factor sign-in for **1 minute**. Each further wrong code doubles the wait, up to **1 hour**. During the wait, sign-in answers "Too many wrong verification codes" (`429`). A right code clears the count. Not configurable.
- **Each code works once.** An authenticator code is accepted a single time, and the next must come from a later 30-second step. After signing in, wait for a new code before the next step that asks for one (such as turning 2FA off). Recovery codes are used up by their first success and stored as SHA-256 hashes.
