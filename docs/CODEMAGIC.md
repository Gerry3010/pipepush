# Codemagic integration

Codemagic (Flutter / iOS / Android) has no per-step "job status" variable in the
build steps themselves. The reliable place to notify is the **`publishing:`
`scripts:`** section: those run **after the build regardless of outcome**, and
expose **`$CM_BUILD_STEP_STATUS`** (`success` / `failure`). That makes it the
right hook to notify on both success and failure.

## 1. Add the token + server as environment variables

In the Codemagic UI: **App settings → Environment variables**, add

- `PIPEPUSH_TOKEN` — the `pp_…` token (mark it **Secure**)
- `PIPEPUSH_SERVER` — e.g. `https://pipepush.geraldhofbauer.net`

Put them in a group (e.g. `pipepush`) and reference the group from the workflow's
`environment.groups:`. A **project-wide** token is easiest here — the `pipeline`
field in the request selects/creates the pipeline by name (use the workflow id).

## 2. Add the notify step to `codemagic.yaml`

Add to each workflow's `publishing:` section (create `publishing:` if absent):

```yaml
workflows:
  ios-testflight-external:
    # environment: { groups: [ pipepush, ... ] }
    # scripts: [ ... your build steps ... ]
    # artifacts: [ ... ]
    publishing:
      scripts:
        - name: Notify pipepush
          script: |
            if [ "$CM_BUILD_STEP_STATUS" = "success" ]; then STATUS=success; else STATUS=failure; fi
            curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \
              -H "Content-Type: application/json" \
              -d "{\"token\":\"$PIPEPUSH_TOKEN\",\"status\":\"$STATUS\",\"pipeline\":\"$CM_WORKFLOW_ID\",\"branch\":\"${CM_BRANCH:-$CM_TAG}\",\"commit\":\"$CM_COMMIT\",\"runId\":\"$BUILD_NUMBER\"}"
      # … your existing app_store_connect / google_play blocks stay here …
```

Codemagic variables used:

| Field   | Variable                | Notes                                  |
|---------|-------------------------|----------------------------------------|
| status  | `$CM_BUILD_STEP_STATUS` | `success` / `failure`                  |
| pipeline| `$CM_WORKFLOW_ID`       | the workflow id (routes/creates by name) |
| branch  | `$CM_BRANCH` / `$CM_TAG`| tag-triggered builds have no branch    |
| commit  | `$CM_COMMIT`            | commit SHA                             |
| runId   | `$BUILD_NUMBER`         | Codemagic build number                 |

## Notes

- Keep it in `publishing:` — a script placed in the build `scripts:` list would be
  **skipped** once an earlier build step fails, so failures (the important case)
  would never notify.
- The contract is identical to every other provider: `POST $PIPEPUSH_SERVER/api/webhook`
  with `{ token, status, pipeline?, branch?, commit?, runId? }`. `status` is
  normalized server-side, so `success`/`failure` are fine.
- Codemagic builds are Apple/Flutter CI — verify the wiring with a real Codemagic
  build (the Mac side of the team) before relying on it.
