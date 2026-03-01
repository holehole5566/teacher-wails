<script lang="ts">
  export let students: Array<{ seat_number: number; name: string }>;
</script>

<div class="card duty-card">
  <div class="card-header">
    <span class="card-icon">🧹</span>
    <h3 class="card-title">今日值日生</h3>
  </div>
  <div class="card-body">
    {#if students && students.length > 0}
      <div class="student-list">
        {#each students as student}
          <div class="student-chip">
            <span class="seat">{student.seat_number}號</span>
            <span class="name">{student.name}</span>
          </div>
        {/each}
      </div>
    {:else}
      <p class="no-data">今日無值日</p>
    {/if}
  </div>
</div>

<style>
  .duty-card {
    border-left: 4px solid var(--accent);
  }
  .card-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
  }
  .card-icon {
    font-size: 20px;
  }
  .card-title {
    font-size: 15px;
    font-weight: 600;
  }
  .student-list {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
  }
  .student-chip {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #eff6ff;
    border-radius: 10px;
    padding: 12px 18px;
  }
  .seat {
    font-size: 13px;
    color: var(--accent);
    font-weight: 600;
  }
  .name {
    font-size: 18px;
    font-weight: 700;
    color: var(--text-primary);
  }
  .no-data {
    color: var(--text-secondary);
    font-size: 14px;
  }
</style>
