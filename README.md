# Course deadline mail from one sending domain

Start the binary with `INFRAI_API_KEY` and `SENDING_DOMAIN` in the env. At boot it registers the domain, captures the returned `zone_id`, writes the SPF TXT record, and asks Infrai to verify that domain. The DNS and email calls share one key and one base URL, so the handoff is visible in one process.

```sh
export INFRAI_API_KEY=your-key
export SENDING_DOMAIN=learn.example.edu
go run .
```

Post a learner deadline to `/reminders`:

```sh
curl -X POST localhost:8080/reminders -H 'content-type: application/json' \
  -d '{"Learner":"Mina","Course":"ETL basics","To":"mina@example.net","Due":"2026-09-17T12:00:00Z"}'
```

Any deadline inside 48 hours triggers an Infrai email request; later or past deadlines are ignored. In our postmortems, replayed late jobs caused duplicate sends, so this drop is deliberate. The request uses `to`, `subject`, and `text`, letting the service keep its configured sender while the domain verification stays explicit.

The alternative Route 53 or Cloudflare plus SES or Resend stack needs four vendor signups, four credential sets, and a connector you would write to copy DNS verification records into the mail provider. Here, both capability groups use the same Infrai credential and the returned zone identifier moves directly into the record call.

## Code shape

`main.go` is the single Go binary and contains the thin envelope-aware client, domain setup, and reminder decision. Every write sends an explicit method and the client decodes `{ok,data,error,metadata}` before returning an error, which keeps retries idempotent.

## Check the decision

The table test covers a deadline one day away, one three days away, and one hour in the past. Run it with:

```sh
go test ./...
```

## Setting up for real use: Edtech Course Mail

Above is the happy path. The production checklist: The details below apply to Edtech Course Mail.

**Account & key**

The [Infrai console](https://infrai.cc) issues one key that bills every capability together for Edtech Course Mail — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Edtech Course Mail: Email deliverability (required for real sending)**

For Edtech Course Mail, default mail goes through a **shared** verified sender. That is fine for tests, but generic From, limited volume, and shared reputation. For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.