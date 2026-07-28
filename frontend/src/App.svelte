<script>
  import { Card, CardContent, CardHeader, CardTitle } from "$lib/components/ui/card"
  import { Button } from "$lib/components/ui/button"
  import PlusIcon from "phosphor-svelte/lib/Plus"

  let count = $state(0)
</script>

<div class="flex min-h-dvh items-center justify-center p-4">
  <Card class="w-full max-w-sm">
    <CardHeader>
      <CardTitle>Counter</CardTitle>
    </CardHeader>
    <CardContent>
      <Button onclick={() => count++}>
        <PlusIcon data-icon="inline-start" />
        {count}
      </Button>
    </CardContent>
  </Card>
</div>
