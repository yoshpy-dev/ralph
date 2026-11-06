# `CLAUDE_CODE_MAX_RETRIES` の回数と待ちの時間(2026-11-07)

計画 `docs/plans/active/2026-11-07-org-seat-resume.md` の S2(AC-11)の記録。
`[org].api_wait_minutes` の分を、claude の座席の `CLAUDE_CODE_MAX_RETRIES` に
換算する式の根拠である。

## 測り方

macOS、Claude Code 2.1.312。名前解決できない宛先
(`ANTHROPIC_BASE_URL=https://ralph-probe.invalid`)に向けて、
`CLAUDE_CODE_RETRY_WATCHDOG=1` と `CLAUDE_CODE_MAX_RETRIES=<N>` を付けた
`claude -p --model haiku "say hi"` を `</dev/null` で打ち、終了までの秒数を
測った。N が 10、20、40 の 3 本を同時に走らせた。どれも最後に
`API Error: Can't reach the API server — check your internet or DNS (ENOTFOUND)`
を出して終了コード 1 で終わった。

| `CLAUDE_CODE_MAX_RETRIES` | 終了までの秒数 |
|---|---|
| 10 | 189 |
| 20 | 554 |
| 40 | 1265 |

参考に、watchdog なしの既定(10 回)は 183 秒だった
(`docs/evidence/org-seat-resume-probe-2026-11-07.md` の 1)。

## 換算の式

10 回目までの立ち上がりが約 189 秒で、そのあとは 1 回あたり約 36 秒かかる
(10 → 20 回で 36.5 秒、20 → 40 回で 35.6 秒)。そこで、`api_wait_minutes` の
分 m から回数 N を次の式で決める。

```text
N = max(1, 10 + round((m × 60 − 189) / 36))
```

| m(分) | N | 見込みの秒数 |
|---|---|---|
| 1 | 6 | 立ち上がりの途中(約 60 秒) |
| 2 | 8 | 立ち上がりの途中(約 120 秒) |
| 3 | 10 | 189 |
| 30(既定) | 55 | 約 1800 |
| 60 | 105 | 約 3600 |

m が 3 分未満のときは、立ち上がりの指数的な待ちの途中で終わるので、式は
おおよその値になる。

## この式が当てはまらないもの

- 応答が返らない通信断。Claude Code は 1 回の再試行で応答のヘッダーを
  `API_TIMEOUT_MS`(既定 10 分)近く待つので、合計は式の時間より長くなりうる
  (公式の errors のページの「No response from API」)。
- 上限と混雑(429/529)。これは `CLAUDE_CODE_MAX_RETRIES` ではなく
  `CLAUDE_CODE_RETRY_WATCHDOG_MAX_WAIT_MS`(m × 60000)で区切られる。
- Claude Code の別の版。再試行の間隔は版で変わりうるので、Claude Code を
  大きく更新したら同じ方法で測り直す。
