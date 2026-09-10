# next

`taskman next <id>` loads the task and asks the pure compiled workflow for its
typed current instruction. This slice only renders that instruction into agent
guidance and a report command; it does not infer workflow position from logs or
timestamps. The same response is available through MCP.

At `specification_review`, it returns a `wait` instruction containing the
drafted specification, acceptance criteria, and the required human approval or
rejection command. After a rejection, the next `specify` dispatch includes the
human's feedback.
