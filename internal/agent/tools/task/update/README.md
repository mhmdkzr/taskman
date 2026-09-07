# task_update

Updates an active task by UUID. The complete task definition and metadata are
provided, and the supplied labels replace the existing labels. `failure_reason`
records why a task failed (for example rejected review feedback or a processing
error) and is overwritten by updates like every other field; omit it to clear.
