# Photo and file questions (F86)

Member app decisions 317, 318 and 320 (2026-10-22). Takes up F43 and F53.

## The answer types

`photo` and `file` join the `answer_type` enum. Their answer fills
`file_ids`, a list of the ids `mwanachama-backend-files` holds. This module
does not import files and never reads a file: an id is an opaque string, and
who may see the file behind it is the host's question (decision 209).

An answer must carry at least one id, with no id empty or given twice.

The other way round holds too: an answer to any question that is not a
`photo` or `file` question is refused when it carries `file_ids`, whatever
else it fills. Ids sent beside a yes, a number or a text would otherwise be
stored with nothing reading them, and a host would grant a reader the file on
the strength of an answer that never asked for one. The refusal sits at the
top of `models.ValidateAnswer`, before the per-type check, so it reaches
every send the same way the value checks do.

## How many

`question.max_files` (nullable) is the question's own limit. Absent means the
host's limit, which the host sets once with `UseDefaultMaxFiles(n)` from the
files module's settings (FL3, photos per question, 5 to start, decision
318). A question's own limit outranks the host's. Neither set means no limit.
`max_files` is refused on a question that does not hold files, and below 1.

`ValidateAnswer` stays as it was and applies no host default;
`ValidateAnswerWithin(q, options, a, defaultMaxFiles)` is the same check with
the host's limit, and is what `SubmitAnswers` calls.

## Taken now, with time and place

`question.require_capture` is a flag a `photo` question may set (decision
320); it is refused on any other type. Forms cannot read a photo's capture
data, so the answer carries the verdict: `captured_file_ids` lists which of
its `file_ids` the files module verified as taken now, with a time and place.
A `require_capture` answer is refused, naming the first file, when any of its
`file_ids` is missing from that list.

`captured_file_ids` is the host's to fill from the files module's record,
never the person answering. The host must overwrite whatever a request body
sent there before the answer reaches `SubmitAnswers`; this module stores what
it is given.
