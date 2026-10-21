# Who collected a send, and through which channel (F36)

Member app decisions 240 and 242. A `submission` carries three fields beside
who sent and when:

| Field | Meaning |
| --- | --- |
| `collector` | Who collected the answers. On a member's own send it is the sender. |
| `collected_at` | When the answers were collected. A paper sheet is collected before it is typed in, so this may be earlier than `submitted_at`. |
| `channel` | `app`, `link`, `field` or `paper`, declared as an enum in the blueprint. |

## What is stamped today

`SubmitAnswers` is the only path that writes a submission, and its caller is
the member answering for themselves, so every send records the sender as
collector and the send time as `collected_at`. The channel is `link` when the
sender reached the form through a public link they claimed or declared under
(the same `cameByLink` test `ReachedOnly` uses, F24/F29), and `app` otherwise. A resend
moves both to the resend, as `edited_at` moves. None of the three is read from
a request.

A send recorded before these fields existed, or backfilled by `Provision` from
older answers, carries all three empty, which reads as "not recorded".

## Not built yet

- A collector answering on another person's behalf (`field`, and `paper`
  typed in later with its own `collected_at`), gated by the PM20 grant, with
  the collector bound `from: caller` and the respondent named in the body.
- Allowing interviewer collection on a member-audience form
  (`ErrAudienceConflict` still refuses it).
