# GitHub Actions integration

## Token scope: pipeline-bound vs project-wide

`pipepush tokens create` makes one of two kinds of token. After creation the CLI
prints a setup snippet tailored to the scope you chose.

- **Pipeline-bound** (`--project … --pipeline …`): every run from this token lands
  in that one pipeline. The `pipeline` field in the request is informational
  (shown in the notification). Best when one workflow ↔ one pipeline.
- **Project-wide** (`--project …`, no `--pipeline`): a single token for the whole
  repo. Each request is routed to a pipeline by its `pipeline` name — and the
  pipeline is **auto-created** on first use. The `pipeline` field is therefore
  **required**. Set it to `${{ github.workflow }}` and each workflow gets its own
  pipeline automatically. Names are matched case-insensitively after trimming.

  > Routing matches on a hash of the (normalized) pipeline name, not the
  > encrypted name. The hash of a low-entropy name like "CI" is not secret, so a
  > project-wide token trades a little pipeline-name privacy for one-token
  > convenience. Use pipeline-bound tokens if that matters to you.

## 1. Add secrets/variables

```bash
gh secret set PIPEPUSH_TOKEN                       # the pp_… token (once)
gh variable set PIPEPUSH_SERVER --body "https://pipepush.example.com"
```

## 2a. Minimal — `curl` (recommended, no binary)

Add to the end of your job:

```yaml
    steps:
      # … your build/test/deploy steps …

      - name: Notify pipepush
        if: always()                 # notify on success AND failure
        run: |
          curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" \
            -H "Content-Type: application/json" \
            -d "{
              \"token\": \"$PIPEPUSH_TOKEN\",
              \"status\": \"${{ job.status }}\",
              \"pipeline\": \"${{ github.workflow }}\",
              \"branch\": \"${{ github.ref_name }}\",
              \"commit\": \"${{ github.sha }}\",
              \"runId\": \"${{ github.run_number }}\"
            }"
        env:
          PIPEPUSH_TOKEN: ${{ secrets.PIPEPUSH_TOKEN }}
          PIPEPUSH_SERVER: ${{ vars.PIPEPUSH_SERVER }}
```

`${{ job.status }}` is `success`, `failure`, or `cancelled` — all understood by pipepush.

## 2b. Using the binary (status normalization, terser command)

```yaml
      - name: Notify pipepush
        if: always()
        run: |
          curl -sL https://github.com/Gerry3010/pipepush/releases/latest/download/pipepush-linux-amd64 \
            -o /usr/local/bin/pipepush && chmod +x /usr/local/bin/pipepush
          pipepush send \
            --token "$PIPEPUSH_TOKEN" \
            --status "${{ job.status }}" \
            --pipeline "${{ github.workflow }}" \
            --branch "${{ github.ref_name }}" \
            --commit "${{ github.sha }}" \
            --run-id "${{ github.run_number }}"
        env:
          PIPEPUSH_TOKEN: ${{ secrets.PIPEPUSH_TOKEN }}
          PIPEPUSH_SERVER: ${{ vars.PIPEPUSH_SERVER }}
```

## 2c. With build logs

Logs are stored end-to-end encrypted (like every other run detail) and shown in the
run detail view; the server keeps the last 16 KB. Pipe the build step's output to a
file, then attach the tail. `jq` is preinstalled on GitHub-hosted runners and escapes
multi-line log text safely:

```yaml
      - name: Build
        run: your-build-command 2>&1 | tee build.log   # capture output

      - name: Notify pipepush
        if: always()
        run: |
          LOGS=$(tail -c 16000 build.log 2>/dev/null || true)
          jq -nc --arg token "$PIPEPUSH_TOKEN" --arg status "${{ job.status }}" \
            --arg pipeline "${{ github.workflow }}" --arg branch "${{ github.ref_name }}" \
            --arg commit "${{ github.sha }}" --arg runId "${{ github.run_number }}" --arg logs "$LOGS" \
            '{token:$token,status:$status,pipeline:$pipeline,branch:$branch,commit:$commit,runId:$runId,logs:$logs}' \
            | curl -sf -X POST "$PIPEPUSH_SERVER/api/webhook" -H "Content-Type: application/json" -d @-
        env:
          PIPEPUSH_TOKEN: ${{ secrets.PIPEPUSH_TOKEN }}
          PIPEPUSH_SERVER: ${{ vars.PIPEPUSH_SERVER }}
```

Or with the binary (no jq): add `--logs-file build.log` to the `pipepush send` in 2b.

## Notify as a separate job (covers the whole workflow)

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps: [ ... ]

  notify:
    needs: [build]
    if: always()
    runs-on: ubuntu-latest
    steps:
      - name: Notify pipepush
        run: |
          curl -sf -X POST "${{ vars.PIPEPUSH_SERVER }}/api/webhook" \
            -H "Content-Type: application/json" \
            -d "{\"token\":\"${{ secrets.PIPEPUSH_TOKEN }}\",\"status\":\"${{ needs.build.result }}\",\"pipeline\":\"${{ github.workflow }}\",\"branch\":\"${{ github.ref_name }}\",\"commit\":\"${{ github.sha }}\"}"
```

`needs.<job>.result` is `success` / `failure` / `cancelled` / `skipped`.
