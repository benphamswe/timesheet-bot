# Runbook: get or refresh the Mavenlink credentials

`submit` needs one Mavenlink secret when `submit_targets` has `mavenlink`. Which one depends on `mavenlink.auth` in `config.yaml` ([configuration](../configuration.md), [ADR-0010](../../adr/0010-add-mavenlink-as-a-second-submit-target.md)). Put the value in `services/timesheet-core/.env` (owner-only, gitignored) or export it in `~/.zshrc`. For the container, a value kept only in `~/.zshrc` needs `docker compose run -e MAVENLINK_SESSION_COOKIE app ...`. Never paste it into chat, a ticket, a commit, or any other file.

## Bearer token (`mavenlink.auth: bearer`, the documented mode)

Only an account administrator can register the OAuth app.

1. Ask an administrator to open Settings > API in Mavenlink and register an application (any name and callback URL; the token is not obtained through the callback).
2. On the registered application's page, they or you click "Show your OAuth Token" while signed in as yourself. The token acts with your own permissions.
3. Set `MAVENLINK_API_TOKEN` to it. Tokens do not expire. Revoke and replace it from the same page if it leaks.

## Session cookie (`mavenlink.auth: cookie`, the fallback)

Use this when no administrator will register an app. Kantata does not document or support it, and the cookie stops working when your Microsoft Entra SSO session ends.

1. Sign in to `https://27global.mavenlink.com` in your browser and open the Timesheets page.
2. Open DevTools (F12) > Network, then reload or change the week, so the page calls the API.
3. Click any request to `/api/v1/...` (for example `time_entries.json`). Under Request Headers, copy the **whole** value of the `Cookie` header.
   - Copy all of it. The cookie names carry an instance suffix (such as `_mavenlink_session_<n>`), and the API needs several of them together, so never pick single cookies.
   - Note whether the request also has an `X-CSRF-Token` header. The bot reads its own token from the page; if the header is missing in the browser, tell the maintainer, because the CSRF assumption in ADR-0010 may be wrong.
4. Set `MAVENLINK_SESSION_COOKIE` to the copied value in `services/timesheet-core/.env`, on one line. If you use `.env`, quote it with single quotes, because the value contains `;` and `=`.
5. Run `timesheet submit --week <week>`.

### When the session expires

`submit` stops with `mavenlink session cookie expired: copy a fresh Cookie header into MAVENLINK_SESSION_COOKIE`. No entry is marked as sent or failed. Repeat steps 1 to 4, then run `submit` again; entries already created in Mavenlink are found by their external reference and not created twice.

### When it reports a CSRF rejection

The page at `mavenlink.csrf_page_url` had no `<meta name="csrf-token">` tag, and Mavenlink answered 403 or 422. Check that the URL opens the Timesheets page while signed in, and compare with the `X-CSRF-Token` header you noted in step 3.

## Rotate

If either value ever appears in chat, a log or a commit, treat it as leaked. For a token, revoke it on the app's page and create a new one. For a cookie, sign out of Mavenlink, which ends the session, then sign in and copy a fresh header.
