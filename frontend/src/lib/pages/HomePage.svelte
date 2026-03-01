<script lang="ts">
  import { onMount } from 'svelte';
  import DutyCard from '../components/DutyCard.svelte';
  import LunchCard from '../components/LunchCard.svelte';
  import { GetTodayDuty } from '../../../wailsjs/go/main/App';

  let displayDate = '';
  let isWorkday = false;
  let dutyStudents: any[] = [];
  let lunchAssignments: any[] = [];
  let loading = true;

  export async function refresh() {
    loading = true;
    try {
      const result = await GetTodayDuty();
      displayDate = result.displayDate || '';
      isWorkday = result.isWorkday || false;
      dutyStudents = result.dutyStudents || [];
      lunchAssignments = result.lunchAssignments || [];
    } catch (e) {
      console.error('Failed to load duty:', e);
    }
    loading = false;
  }

  onMount(refresh);
</script>

<div class="page">
  <div class="date-header">
    <h2 class="page-title">今日值日</h2>
    <span class="date-display">{displayDate}</span>
  </div>

  {#if loading}
    <div class="empty-state">載入中...</div>
  {:else if !isWorkday}
    <div class="card">
      <div class="rest-message">
        <span class="rest-icon">🎉</span>
        <h3>今日休息</h3>
        <p>今天是週末或假期，無需值日</p>
      </div>
    </div>
  {:else}
    <div class="cards-grid">
      <DutyCard students={dutyStudents} />
      <LunchCard assignments={lunchAssignments} />
    </div>
  {/if}
</div>

<style>
  .date-header {
    display: flex;
    align-items: baseline;
    gap: 16px;
    margin-bottom: 20px;
  }
  .date-display {
    color: var(--text-secondary);
    font-size: 14px;
  }
  .cards-grid {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .rest-message {
    text-align: center;
    padding: 40px 20px;
  }
  .rest-icon {
    font-size: 48px;
    display: block;
    margin-bottom: 12px;
  }
  .rest-message h3 {
    font-size: 18px;
    margin-bottom: 8px;
  }
  .rest-message p {
    color: var(--text-secondary);
  }
</style>
