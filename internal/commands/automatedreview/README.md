# automated-review

`taskman automated-review approved <id>` records a successful independent
automated review. `taskman automated-review rejected <id> --finding
file=detail` records its findings. The first rejection routes to fixes and
verification; the second blocks the task.
