<script lang="ts">
  import { onMount } from 'svelte';
  import { GetSettings, SaveSettings } from '../../../wailsjs/go/main/App';

  let semesterStart = '';
  let dutyGroupSize = 2;
  let dutyStartNumber = 1;
  let lunchGroupSize = 5;
  let lunchStartNumber = 1;
  let mealBucketsStr = '';
  let saved = false;

  async function loadSettings() {
    const s = await GetSettings();
    semesterStart = s.semester_start_date || '';
    dutyGroupSize = s.duty_group_size || 2;
    dutyStartNumber = s.duty_start_number || 1;
    lunchGroupSize = s.lunch_group_size || 5;
    lunchStartNumber = s.lunch_start_number || 1;
    mealBucketsStr = (s.meal_buckets || []).join('，');
  }

  async function handleSave() {
    const buckets = mealBucketsStr
      .split(/[,，]/)
      .map((b: string) => b.trim())
      .filter((b: string) => b);

    await SaveSettings({
      semester_start_date: semesterStart,
      duty_group_size: dutyGroupSize,
      duty_start_number: dutyStartNumber,
      lunch_group_size: lunchGroupSize,
      lunch_start_number: lunchStartNumber,
      meal_buckets: buckets,
      auto_start: false,
    });
    saved = true;
    setTimeout(() => { saved = false; }, 2000);
  }

  onMount(loadSettings);
</script>

<div class="page">
  <h2 class="page-title">設定</h2>

  <div class="card settings-form">
    <div class="form-group">
      <label>學期開始日期</label>
      <input type="date" bind:value={semesterStart} />
    </div>

    <div class="form-row">
      <div class="form-group">
        <label>值日每組人數</label>
        <input type="number" min="1" bind:value={dutyGroupSize} />
      </div>
      <div class="form-group">
        <label>值日起始座號</label>
        <input type="number" min="1" bind:value={dutyStartNumber} />
      </div>
    </div>

    <div class="form-row">
      <div class="form-group">
        <label>抬餐每組人數</label>
        <input type="number" min="1" bind:value={lunchGroupSize} />
      </div>
      <div class="form-group">
        <label>抬餐起始座號</label>
        <input type="number" min="1" bind:value={lunchStartNumber} />
      </div>
    </div>

    <div class="form-group">
      <label>餐桶名稱（以逗號分隔）</label>
      <input type="text" bind:value={mealBucketsStr} placeholder="飯，菜1，菜2，湯，餐具" />
    </div>

    <div class="form-actions">
      <button class="btn-primary" on:click={handleSave}>儲存設定</button>
      {#if saved}
        <span class="save-ok">已儲存</span>
      {/if}
    </div>
  </div>
</div>

<style>
  .settings-form {
    max-width: 480px;
  }
  .form-row {
    display: flex;
    gap: 16px;
  }
  .form-row .form-group {
    flex: 1;
  }
  .form-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 8px;
  }
  .save-ok {
    color: var(--success);
    font-size: 13px;
    font-weight: 500;
  }
</style>
