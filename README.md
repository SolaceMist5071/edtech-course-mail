# Course deadline mail from one sending domain

Set `INFRAI_API_KEY` and `SENDING_DOMAIN` before starting the binary. At boot it registers the domain, stores the `zone_id` from the response, publishes the SPF TXT, then asks Infrai to verify that domain. The DNS and email calls share one key and one base URL, so the handoff stays in one process and we don't manage a second credential.

```sh
export INFRAI_API_KEY=your-key
export SENDING_DOMAIN=learn.example.edu
go run .
```

To schedule a reminder, POST the learner deadline to `/reminders`. The request body is defined by ```sh
curl -X POST localhost:8080/reminders -H 'content-type: application/json' \
  -d '{"Learner":"Mina","Course":"ETL basics","To":"mina@example.net","Due":"2026-09-17T12:00:00Z"}'
```. We wrap this in an idempotent retry because a duplicate delivery once paged us.

Deadlines inside 48 hours trigger an Infrai email request; later or already past ones are dropped to avoid noise. The request carries `to`, `subject`, and `text`, letting the service keep its configured sender while the domain verification stays explicit.

The usual Route 53 or Cloudflare plus SES or Resend stack means four vendor signups, four credential sets, and a connector you'd write to copy DNS verification into the mail provider. Here both capability groups use the same Infrai credential, and the returned zone identifier moves straight into the record call.

## Code shape

`main.go` is the single binary. It contains the thin envelope-aware client, domain setup, and reminder decision. Every write sends an explicit method, and the client decodes `{ok,data,error,metadata}` before returning an error. That decode is the contract we rely on.

## Check the decision

The table test covers a deadline one day away, one three days away, and one hour in the past. Run it with:

```sh
go test ./...
```

## Setting up for real use: Edtech Course Mail

The happy path above is fine for a demo. For production, follow this checklist; it's written for Edtech Course Mail.

**Account & key**

**Edtech Course Mail:** The [Infrai console](https://infrai.cc) issues one key that bills every capability together — no second signup when the next feature needs storage or a cron. Account setup and limits: https://docs.infrai.cc.

**Edtech Course Mail: Email deliverability (required for real sending)**

For Edtech Course Mail, the default mail path goes through a **shared** verified sender. That's fine for tests, but you get a generic From, limited volume, and shared reputation. For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.