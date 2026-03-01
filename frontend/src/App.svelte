<script lang="ts">
  import './style.css';
  import Sidebar from './lib/components/Sidebar.svelte';
  import HomePage from './lib/pages/HomePage.svelte';
  import StudentsPage from './lib/pages/StudentsPage.svelte';
  import SettingsPage from './lib/pages/SettingsPage.svelte';
  import HolidaysPage from './lib/pages/HolidaysPage.svelte';
  import { ExportSchedule } from '../wailsjs/go/main/App';

  let currentPage = 'home';
  let homeRef: HomePage;

  async function handleNavigate(page: string) {
    if (page === 'export') {
      try {
        const path = await ExportSchedule();
        if (path) {
          alert(`已匯出至: ${path}`);
        }
      } catch (e) {
        alert('匯出失敗: ' + e);
      }
      return;
    }
    currentPage = page;
    // Refresh home page when navigating back to it
    if (page === 'home' && homeRef) {
      homeRef.refresh();
    }
  }
</script>

<Sidebar {currentPage} onNavigate={handleNavigate} />

<main class="main-content">
  {#if currentPage === 'home'}
    <HomePage bind:this={homeRef} />
  {:else if currentPage === 'students'}
    <StudentsPage />
  {:else if currentPage === 'settings'}
    <SettingsPage />
  {:else if currentPage === 'holidays'}
    <HolidaysPage />
  {/if}
</main>

<style>
  .main-content {
    flex: 1;
    display: flex;
    overflow: hidden;
  }
</style>
