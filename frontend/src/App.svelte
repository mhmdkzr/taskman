<script>
  let count = $state(0)
</script>

<main>
  <h1>Hello, world!</h1>
  <button onclick={() => count++}>
    Clicked {count} {count === 1 ? 'time' : 'times'}
  </button>
</main>

<style>
  main {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 2rem;
    font-family: system-ui, sans-serif;
  }
  h1 {
    color: #333;
  }
  button {
    padding: 0.5rem 1rem;
    font-size: 1rem;
    cursor: pointer;
  }
</style>
