<!--
SPDX-FileCopyrightText: 2026 Chen Linxuan <me@black-desk.cn>

SPDX-License-Identifier: MIT
-->

1. For every commit assisted by an LLM, follow the transparency guidance in the
   Linux kernel's `Documentation/process/generated-content.rst` and
   `Documentation/process/coding-assistants.rst`:
   - Describe the problem, the solution, and the validation in the commit
     message. Explicitly state any checks that could not be completed.
   - Disclose which AI tools were used and which portions of the contribution
     they affected, including problem discovery, code, documentation, commit
     message generation or translation, and testing.
   - Include the prompts when the contribution was largely generated from a
     single prompt or a short set of prompts. For longer sessions, summarize the
     prompts and the nature of the assistance.
   - AI agents must not add `Signed-off-by` trailers. Only human submitters can
     certify the Developer Certificate of Origin; they are responsible for
     reviewing and understanding the contribution and adding their own sign-off
     when applicable.

2. Before making any changes, read the project's README.md and CONTRIBUTING.md.
