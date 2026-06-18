# GitLab CI integration

## 1. Add CI/CD variables

Project → **Settings → CI/CD → Variables**, add (both **masked**):

- `PIPEPUSH_TOKEN` = your `pp_…` token
- `PIPEPUSH_SERVER` = `https://pipepush.example.com`

## 2. Add a notification job

```yaml
notify_pipepush:
  stage: .post              # runs after everything else
  when: always              # notify on success AND failure
  image: curlimages/curl:latest
  script:
    - |
      curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \
        -H "Content-Type: application/json" \
        -d "{
          \"token\": \"$PIPEPUSH_TOKEN\",
          \"status\": \"$CI_JOB_STATUS\",
          \"pipeline\": \"$CI_PIPELINE_NAME\",
          \"branch\": \"$CI_COMMIT_REF_NAME\",
          \"commit\": \"$CI_COMMIT_SHA\",
          \"runId\": \"$CI_PIPELINE_IID\"
        }"
```

Notes:

- `$CI_JOB_STATUS` is `success` / `failed` / `canceled` — pipepush normalizes
  `failed` → `failure` and `canceled` → `cancelled` automatically.
- The `.post` stage exists by default; no need to declare it in `stages:`.
- To report the **whole pipeline's** result rather than this job's, use a
  `rules`-driven job or the [pipeline status webhook](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)
  pointed at `$PIPEPUSH_SERVER/api/webhook` with a JSON body containing your token.
