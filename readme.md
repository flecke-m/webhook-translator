# webhook-translator

A lightweight Go service that translates **Unifi Protect doorbell webhooks** into [ntfy](https://ntfy.sh) push notifications with an attached snapshot image.

## How it works

When a Unifi Protect doorbell rings it sends an HTTP webhook containing a JSON payload with alarm details and a base64-encoded JPEG snapshot (`alarm.thumbnail`). This service receives that webhook, extracts the snapshot, and forwards it as a binary PUT request to an ntfy topic — resulting in a push notification with the doorbell image attached.

Configuration is passed entirely via HTTP request headers (no secrets in the body):

| Header | Description |
|---|---|
| `Authorization` | ntfy authentication token, e.g. `Bearer tk_...` — forwarded as-is |
| `topic` | ntfy topic to publish to |
| `Title` | Notification title (e.g. `Es klingelt`) |
| `Tags` | Comma-separated ntfy tags (e.g. `door,bell`) |
| `ntfy_url` | Base URL of your ntfy instance (default: `https://ntfy.sh`) |

## Running with Docker

### Build

```sh
docker build -t webhook-translator .
```

### Run

```sh
docker run -p 80:80 webhook-translator
```

The service listens on port **80** inside the container.

### Example request

```sh
curl -X POST http://localhost:80 \
  -H "Authorization: Bearer tk_yourtoken" \
  -H "Title: Es klingelt" \
  -H "Tags: door,bell" \
  -H "ntfy_url: https://ntfy.sh" \
  -H "topic: your-topic" \
  -H "Content-Type: application/json" \
  -d '{ "alarm": { "thumbnail": "data:image/jpeg;base64,..." } }'
```
