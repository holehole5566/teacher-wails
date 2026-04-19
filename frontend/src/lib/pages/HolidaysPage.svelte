<script lang="ts">
  import { onMount } from 'svelte';
  import { GetHolidays, AddHoliday, DeleteHoliday, ClearHolidays, SyncHolidays } from '../../../wailsjs/go/main/App';

  let holidays: string[] = [];
  let newDate = '';
  let syncing = false;

  async function loadHolidays() {
    holidays = (await GetHolidays()) || [];
    holidays.sort();
  }

  async function handleAdd() {
    if (!newDate) return;
    await AddHoliday(newDate);
    newDate = '';
    await loadHolidays();
  }

  async function handleDelete(dateStr: string) {
    await DeleteHoliday(dateStr);
    await loadHolidays();
  }

  async function handleClear() {
    await ClearHolidays();
    await loadHolidays();
  }

  async function handleSync() {
    syncing = true;
    try {
      const added = await SyncHolidays();
      await loadHolidays();
      alert(`同步完成，新增 ${added} 筆假期`);
    } catch (e) {
      alert('同步失敗: ' + e);
    } finally {
      syncing = false;
    }
  }

  onMount(loadHolidays);
</script>

<div class="page">
  <h2 class="page-title">假期管理</h2>

  <div class="card">
    <div class="inline-form" style="margin-bottom: 16px;">
      <input type="date" bind:value={newDate} />
      <button class="btn-primary" on:click={handleAdd}>新增假期</button>
      <button class="btn-primary" on:click={handleSync} disabled={syncing}>
        {syncing ? '同步中...' : '🔄 同步政府假日'}
      </button>
      {#if holidays.length > 0}
        <button class="btn-danger" on:click={handleClear}>清空全部</button>
      {/if}
    </div>

    {#if holidays.length === 0}
      <div class="empty-state">尚無假期</div>
    {:else}
      <div class="holiday-list">
        {#each holidays as h}
          <div class="holiday-item">
            <span class="holiday-date">{h}</span>
            <button class="btn-icon" on:click={() => handleDelete(h)}>✕</button>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>

<style>
  .holiday-list {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  .holiday-item {
    display: flex;
    align-items: center;
    gap: 6px;
    background: var(--bg-primary);
    padding: 6px 10px 6px 14px;
    border-radius: 20px;
    font-size: 13px;
  }
  .holiday-date {
    color: var(--text-primary);
  }
</style>
