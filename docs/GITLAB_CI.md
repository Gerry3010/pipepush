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
      case "$CI_JOB_STATUS" in
        success) STATUS=success ;;
        canceled) STATUS=cancelled ;;
        *) STATUS=failure ;;
      esac
      curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \
        -H "Content-Type: application/json" \
        -d "{
          \"token\": \"$PIPEPUSH_TOKEN\",
          \"status\": \"$STATUS\",
          \"pipeline\": \"$CI_PIPELINE_NAME\",
          \"branch\": \"$CI_COMMIT_REF_NAME\",
          \"commit\": \"$CI_COMMIT_SHA\",
          \"runId\": \"$CI_PIPELINE_IID\"
        }"
```

Notes:

- `$CI_JOB_STATUS` is `success` / `failed` / `canceled`. The webhook accepts only
  `success` / `failure` / `cancelled` / `running` / `skipped` and does **not**
  normalize, so the `case` maps GitLab's values first — otherwise `failed` and
  `canceled` are rejected with HTTP 400 and (because of `curl -sf`) fail silently.
- The `.post` stage exists by default; no need to declare it in `stages:`.
- To report the **whole pipeline's** result rather than this job's, use a
  `rules`-driven job or the [pipeline status webhook](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html)
  pointed at `$PIPEPUSH_SERVER/api/webhook` with a JSON body containing your token.

## 3. Optional: attach build logs

Logs are stored end-to-end encrypted and shown in the run detail view (server keeps
the last 16 KB). The `curlimages/curl` image has no `jq`, so the simplest path is the
`pipepush` binary, which escapes and truncates the log for you. Pipe your build to a
file and persist it as an artifact so the notify job can read it:

```yaml
build:
  script:
    - your-build-command 2>&1 | tee build.log
  artifacts:
    when: always
    paths: [build.log]

notify_pipepush:
  stage: .post
  when: always
  image: alpine:latest
  script:
    - |
      case "$CI_JOB_STATUS" in success) S=success ;; canceled) S=cancelled ;; *) S=failure ;; esac
      wget -qO /usr/local/bin/pipepush "$PIPEPUSH_DOWNLOAD_URL" && chmod +x /usr/local/bin/pipepush
      pipepush send --token "$PIPEPUSH_TOKEN" --status "$S" \
        --pipeline "$CI_PIPELINE_NAME" --branch "$CI_COMMIT_REF_NAME" \
        --commit "$CI_COMMIT_SHA" --logs-file build.log
```

(Or add `jq` to the curl image — `apk add --no-cache jq` — and build the JSON with
`jq -nc --arg logs "$(tail -c 16000 build.log)" …` as in the other providers.)
