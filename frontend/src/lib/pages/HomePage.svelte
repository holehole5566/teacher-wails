<script lang="ts">
  import { onMount } from 'svelte';
  import DutyCard from '../components/DutyCard.svelte';
  import LunchCard from '../components/LunchCard.svelte';
  import StudentPicker from '../components/StudentPicker.svelte';
  import { GetTodayDuty, GetStudents } from '../../../wailsjs/go/main/App';

  let displayDate = '';
  let isWorkday = false;
  let dutyStudents: any[] = [];
  let lunchAssignments: any[] = [];
  let loading = true;

  // Original data from scheduler (for reset)
  let origDuty: any[] = [];
  let origLunch: any[] = [];

  // Manual override tracking
  let dutyModified = false;
  let lunchModified = false;

  // Picker state
  let allStudents: any[] = [];
  let showPicker = false;
  let pickerCallback: ((s: any) => void) | null = null;

  export async function refresh() {
    loading = true;
    try {
      const result = await GetTodayDuty();
      displayDate = result.displayDate || '';
      isWorkday = result.isWorkday || false;
      origDuty = result.dutyStudents || [];
      origLunch = result.lunchAssignments || [];
      dutyStudents = [...origDuty];
      lunchAssignments = origLunch.map((a: any) => ({ ...a }));
      dutyModified = false;
      lunchModified = false;
      allStudents = await GetStudents();
    } catch (e) {
      console.error('Failed to load duty:', e);
    }
    loading = false;
  }

  function openPicker(cb: (s: any) => void) {
    pickerCallback = cb;
    showPicker = true;
  }

  function closePicker() {
    showPicker = false;
    pickerCallback = null;
  }

  function replaceDuty(index: number) {
    openPicker((s) => {
      dutyStudents[index] = s;
      dutyStudents = dutyStudents;
      dutyModified = true;
      closePicker();
    });
  }

  function replaceLunch(index: number) {
    openPicker((s) => {
      lunchAssignments[index] = { ...lunchAssignments[index], student: s };
      lunchAssignments = lunchAssignments;
      lunchModified = true;
      closePicker();
    });
  }

  function resetAll() {
    dutyStudents = [...origDuty];
    lunchAssignments = origLunch.map((a: any) => ({ ...a }));
    dutyModified = false;
    lunchModified = false;
  }

  onMount(refresh);
</script>

{#if showPicker}
  <StudentPicker students={allStudents} onPick={(s) => pickerCallback && pickerCallback(s)} onClose={closePicker} />
{/if}

<div class="page">
  <div class="date-header">
    <h2 class="page-title">今日值日</h2>
    <span class="date-display">{displayDate}</span>
    {#if dutyModified || lunchModified}
      <button class="btn-outline btn-sm" on:click={resetAll}>↺ 重設排程</button>
    {/if}
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
      <DutyCard students={dutyStudents} modified={dutyModified} onReplace={replaceDuty} />
      <LunchCard assignments={lunchAssignments} modified={lunchModified} onReplace={replaceLunch} />
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
